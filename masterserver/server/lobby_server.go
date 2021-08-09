package server

import (
	"context"
	"quark"
	"quark/proto"
	"quark/proto/primitive"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LobbyServer struct {
	proto.UnimplementedLobbyServer

	fleet *quark.Fleet
}

func (s *LobbyServer) CreateRoom(ctx context.Context, req *proto.CreateRoomRequest) (*proto.CreateRoomResponse, error) {
	var roomName string
	if len(req.RoomName) == 0 {
		roomName = "default"
	} else {
		roomName = req.RoomName
	}

	roomID := quark.NewRoomID()
	_, err := s.fleet.AllocateRoom(roomID, roomName)
	if err == nil && err != quark.ErrRoomAlreadyAllocated {
		return nil, status.Errorf(codes.Aborted, err.Error())
	}
	return &proto.CreateRoomResponse{
		RoomID:       roomID.Uint64(),
		AlreadyExist: err == quark.ErrRoomAlreadyAllocated,
	}, nil
}

func (s *LobbyServer) InLobby(req *proto.InLobbyRequest, stream proto.Lobby_InLobbyServer) error {
	c := make(chan quark.RoomAllocatedEvent)
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

func (s *LobbyServer) JoinRoom(ctx context.Context, req *proto.JoinRoomRequest) (*proto.JoinRoomResponse, error) {
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
