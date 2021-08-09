package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"quark"
	"quark/masterserver"
	"quark/proto"
	"quark/proto/primitive"
)

type lobbyServer struct {
	proto.UnimplementedLobbyServer

	fleet *masterserver.Fleet
}

func NewLobbyServer(fleet *masterserver.Fleet) proto.LobbyServer {
	return &lobbyServer{fleet: fleet}
}

func (s *lobbyServer) CreateRoom(ctx context.Context, req *proto.CreateRoomRequest) (*proto.CreateRoomResponse, error) {
	var roomName string
	if len(req.RoomName) == 0 {
		roomName = "default"
	} else {
		roomName = req.RoomName
	}

	roomID := quark.NewRoomID()
	_, err := s.fleet.AllocateRoom(roomID, roomName)
	if err != nil && err != masterserver.ErrRoomAlreadyAllocated {
		return nil, status.Errorf(codes.Aborted, err.Error())
	}
	return &proto.CreateRoomResponse{
		RoomID:       roomID.Uint64(),
		AlreadyExist: err == masterserver.ErrRoomAlreadyAllocated,
	}, nil
}

func (s *lobbyServer) InLobby(req *proto.InLobbyRequest, stream proto.Lobby_InLobbyServer) error {
	c := make(chan masterserver.RoomAllocatedEvent)
	s.fleet.AddRoomAllocationListener(c)
	defer func() {
		s.fleet.RemoveRoomAllocationListener(c)
		close(c)
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-c:
			rs := s.fleet.RoomList()

			roomList := make([]*primitive.Room, len(rs))

			for i, r := range rs {
				roomList[i] = &primitive.Room{
					RoomID:   r.RoomID.Uint64(),
					RoomName: r.RoomName,
				}
			}
			m := &proto.InLobbyMessage{
				Message: &proto.InLobbyMessage_OnUpdatedRoomList{
					OnUpdatedRoomList: &proto.InLobbyMessage_RoomListUpdatedEvent{
						RoomList: roomList,
					},
				},
			}
			err := stream.Send(m)
			if err != nil {
				return err
			}
		}
	}
}

func (s *lobbyServer) JoinRoom(ctx context.Context, req *proto.JoinRoomRequest) (*proto.JoinRoomResponse, error) {
	roomID := quark.RoomID(req.RoomID)
	addr, ok := s.fleet.LookupGameServerAddr(roomID)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room is not found")
	}
	return &proto.JoinRoomResponse{
		Server: &primitive.GameServer{
			Address: addr.Addr,
			Port:    addr.Port,
		},
	}, nil
}
