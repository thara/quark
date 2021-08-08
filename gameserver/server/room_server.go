package server

import (
	"context"
	"io"

	"quark"
	"quark/proto"
)

type roomServer struct {
	proto.UnimplementedRoomServer

	roomSet *quark.RoomSet
}

func NewRoomServer() proto.RoomServer {
	return &roomServer{
		roomSet: quark.NewRoomSet(),
	}
}

func (s *roomServer) CreateRoom(ctx context.Context, req *proto.CreateRoomRequest) (*proto.CreateRoomResponse, error) {
	roomID, loaded := s.roomSet.NewRoom(req.RoomName)
	return &proto.CreateRoomResponse{
		RoomID:       roomID.Uint64(),
		AlreadyExist: loaded,
	}, nil
}

func (s *roomServer) Service(stream proto.Room_ServiceServer) error {
	fail := make(chan error, 1)

	onJoined := make(chan interface{})
	onLeaved := make(chan interface{})
	onCommandFailed := make(chan commandError, 1)

	actor := quark.NewActor()

	// recv loop
	go func() {
		defer close(onJoined)
		defer close(onLeaved)
		defer close(onCommandFailed)

		for {
			select {
			case <-stream.Context().Done():
				actor.Leave()
				return
			default:
				in, err := stream.Recv()
				if err == io.EOF {
					return
				} else if err != nil {
					fail <- err
					return
				}

				switch cmd := in.Command.(type) {
				case *proto.ClientMessage_JoinRoom:
					roomID := quark.RoomID(cmd.JoinRoom.RoomID)
					if ok := s.roomSet.JoinRoom(roomID, actor); ok {
						onJoined <- struct{}{}
					} else {
						onCommandFailed <- commandError{code: "001", detail: "room does not exist", cmd: cmd.JoinRoom}
					}
				case *proto.ClientMessage_SendMessage:
					ok := actor.BroadcastToRoom(quark.Payload{
						Code: cmd.SendMessage.Message.Code,
						Body: cmd.SendMessage.Message.Payload})
					if !ok {
						onCommandFailed <- commandError{code: "001", detail: "room does not exist", cmd: cmd.SendMessage}
					}
				case *proto.ClientMessage_LeaveRoom:
					actor.Leave()
					onLeaved <- struct{}{}
				}
			}
		}
	}()

	// send loop
	go func() {
		inbox := actor.Inbox()

		for {
			select {
			case <-stream.Context().Done():
				return
			case c := <-onCommandFailed:
				e := toCommandErrorEvent(c)
				msg := proto.ServerMessage{
					Event: &proto.ServerMessage_OnCommandFailed{
						OnCommandFailed: e,
					},
				}
				if err := stream.Send(&msg); err != nil {
					fail <- err
				}
			case <-onJoined:
				inbox = actor.Inbox()

				msg := proto.ServerMessage{
					Event: &proto.ServerMessage_OnJoinRoomSuccess{
						OnJoinRoomSuccess: &proto.ServerMessage_JoinRoomSuccess{
							ActorID: actor.ActorID().String(),
						},
					},
				}
				if err := stream.Send(&msg); err != nil {
					fail <- err
				}
			case m, ok := <-inbox:
				if !ok {
					continue
				}
				if actor.IsOwnMessage(&m) {
					// skip send
					continue
				}
				msg := proto.ServerMessage{
					Event: &proto.ServerMessage_OnMessageReceived{
						OnMessageReceived: &proto.ServerMessage_ReceivedMessageEvent{
							SenderID: m.Sender.String(),
							Message: &proto.Message{
								Code:    m.Code,
								Payload: m.Payload,
							},
						},
					},
				}
				if err := stream.Send(&msg); err != nil {
					fail <- err
				}
			case <-onLeaved:
				inbox = actor.Inbox()

				msg := proto.ServerMessage{
					Event: &proto.ServerMessage_OnLeaveRoomSuccess{
						OnLeaveRoomSuccess: &proto.ServerMessage_LeaveRoomSuccess{},
					},
				}
				if err := stream.Send(&msg); err != nil {
					fail <- err
				}
			}
		}
	}()

	// error handling loop
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case err := <-fail:
			return err
		}
	}
}

type commandError struct {
	code   string
	detail string
	cmd    interface{}
}

func toCommandErrorEvent(c commandError) *proto.ServerMessage_CommandError {
	switch cmd := c.cmd.(type) {
	case *proto.ClientMessage_JoinRoomCommand:
		return &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			ErrorCommand: &proto.ServerMessage_CommandError_JoinRoom{
				JoinRoom: cmd,
			},
		}
	case *proto.ClientMessage_SendMessageCommand:
		return &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			ErrorCommand: &proto.ServerMessage_CommandError_SendMessage{
				SendMessage: cmd,
			},
		}
	default:
		return &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
		}
	}
}
