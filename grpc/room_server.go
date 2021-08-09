package grpc

import (
	"context"
	"io"

	"quark"
	"quark/gameserver"
	"quark/proto"
)

type roomServer struct {
	proto.UnimplementedRoomServer

	roomSet *gameserver.RoomSet
}

func NewRoomServer() proto.RoomServer {
	return &roomServer{
		roomSet: gameserver.NewRoomSet(),
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

	actor := gameserver.NewActor()

	// recv loop
	go func() {
		defer close(onJoined)
		defer close(onLeaved)

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
						msg := toServerMessage(commandError{code: "001", detail: "room does not exist", cmd: cmd.JoinRoom})
						if err := stream.Send(msg); err != nil {
							fail <- err
						}
					}
				case *proto.ClientMessage_SendMessage:
					ok := actor.BroadcastToRoom(gameserver.Payload{
						Code: cmd.SendMessage.Message.Code,
						Body: cmd.SendMessage.Message.Payload})
					if !ok {
						msg := toServerMessage(commandError{code: "001", detail: "room does not exist", cmd: cmd.SendMessage})
						if err := stream.Send(msg); err != nil {
							fail <- err
						}
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
			case m, ok := <-inbox:
				if !ok {
					continue
				}

				switch m := m.(type) {
				case gameserver.ActorMessage:
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
				case gameserver.JoinRoomEvent:
					ids := make([]string, len(m.ActorList))
					for i, a := range m.ActorList {
						ids[i] = a.String()
					}
					msg := proto.ServerMessage{
						Event: &proto.ServerMessage_OnJoinRoom{
							OnJoinRoom: &proto.ServerMessage_JoinRoom{
								ActorIDList: ids,
								NewActorID:  m.NewActor.String(),
							},
						},
					}
					if err := stream.Send(&msg); err != nil {
						fail <- err
					}
				case gameserver.LeaveRoomEvent:
					ids := make([]string, len(m.ActorList))
					for i, a := range m.ActorList {
						ids[i] = a.String()
					}
					msg := proto.ServerMessage{
						Event: &proto.ServerMessage_OnLeaveRoom{
							OnLeaveRoom: &proto.ServerMessage_LeaveRoom{
								ActorIDList:    ids,
								RemovedActorID: m.RemovedActor.String(),
							},
						},
					}
					if err := stream.Send(&msg); err != nil {
						fail <- err
					}
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

func toServerMessage(c commandError) *proto.ServerMessage {
	var cmdErr *proto.ServerMessage_CommandError
	switch cmd := c.cmd.(type) {
	case *proto.ClientMessage_JoinRoomCommand:
		cmdErr = &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			ErrorCommand: &proto.ServerMessage_CommandError_JoinRoom{
				JoinRoom: cmd,
			},
		}
	case *proto.ClientMessage_SendMessageCommand:
		cmdErr = &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			ErrorCommand: &proto.ServerMessage_CommandError_SendMessage{
				SendMessage: cmd,
			},
		}
	default:
		cmdErr = &proto.ServerMessage_CommandError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
		}
	}

	return &proto.ServerMessage{
		Event: &proto.ServerMessage_OnCommandFailed{
			OnCommandFailed: cmdErr,
		},
	}
}
