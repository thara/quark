package room

import (
	"context"
	"io"
	"quark"
	"quark/gameserver"
	"sync/atomic"

	"github.com/google/uuid"
)

type roomServer struct {
	gameserver.UnimplementedRoomServer

	roomList *roomList
}

func NewRoomServer() gameserver.RoomServer {
	return &roomServer{
		roomList: newRoomList(),
	}
}

func (s *roomServer) CreateRoom(ctx context.Context, req *gameserver.CreateRoomRequest) (*gameserver.CreateRoomResponse, error) {
	var roomName string
	if len(req.RoomName) == 0 {
		roomName = uuid.Must(uuid.NewRandom()).String()
	} else {
		roomName = req.RoomName
	}
	roomID, loaded := s.roomList.newRoom(roomName)
	return &gameserver.CreateRoomResponse{
		RoomID:       roomID.Uint64(),
		AlreadyExist: loaded,
	}, nil
}

func (s *roomServer) Service(stream gameserver.Room_ServiceServer) error {
	fail := make(chan error, 1)
	cmdFailed := make(chan commandError, 1)

	subscription := make(chan message)
	defer close(subscription)

	actorID := quark.NewActorID()

	var curRoom uint64

	currentRoom := func() (*room, bool) {
		id := atomic.LoadUint64(&curRoom)
		if 0 < id {
			if r, ok := s.roomList.getRoom(roomID(id)); ok {
				return r, true
			}
		}
		return nil, false
	}
	leaveRoom := func() {
		if r, ok := currentRoom(); ok {
			r.leave <- subscription
		}
	}

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				fail <- err
				return
			}

			switch c := in.CommandType.(type) {
			case *gameserver.Command_JoinRoom:
				cmd := c.JoinRoom
				roomID := roomID(cmd.RoomID)
				if r, ok := s.roomList.getRoom(roomID); ok {
					r.Join(subscription)
					atomic.StoreUint64(&curRoom, roomID.Uint64())
				} else {
					cmdFailed <- commandError{code: "001", detail: "room does not exist", cmd: cmd}
				}
			case *gameserver.Command_LeaveRoom:
				if r, ok := currentRoom(); ok {
					r.Leave(subscription)
				}
			case *gameserver.Command_SendMessage:
				cmd := c.SendMessage
				if r, ok := currentRoom(); ok {
					r.Send(message{
						sender:  actorID,
						code:    cmd.Message.Code,
						payload: cmd.Message.Payload,
					})
				} else {
					cmdFailed <- commandError{code: "001", detail: "room does not exist", cmd: cmd}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case m := <-subscription:
				if m.sender == actorID {
					// skip send
					continue
				}
				ev := gameserver.Event{
					EventType: &gameserver.Event_MessageReceived{
						MessageReceived: &gameserver.MessageReceived{
							SenderID: m.sender.String(),
							Message: &gameserver.Message{
								Code:    m.code,
								Payload: m.payload,
							},
						},
					},
				}
				if err := stream.Send(&ev); err != nil {
					fail <- err
				}
			case c := <-cmdFailed:
				e := toCommandOperationError(c)
				ev := gameserver.Event{
					EventType: &gameserver.Event_CommandOperationError{
						CommandOperationError: e,
					},
				}
				if err := stream.Send(&ev); err != nil {
					fail <- err
				}
			}
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			leaveRoom()
			return stream.Context().Err()
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

func toCommandOperationError(c commandError) *gameserver.CommandOperationError {
	switch cmd := c.cmd.(type) {
	case *gameserver.Command_JoinRoom:
		return &gameserver.CommandOperationError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			CommandType: &gameserver.CommandOperationError_JoinRoom{
				JoinRoom: cmd.JoinRoom,
			},
		}
	case *gameserver.Command_SendMessage:
		return &gameserver.CommandOperationError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
			CommandType: &gameserver.CommandOperationError_SendMessage{
				SendMessage: cmd.SendMessage,
			},
		}
	default:
		return &gameserver.CommandOperationError{
			ErrorCode:   c.code,
			ErrorDetail: c.detail,
		}
	}
}
