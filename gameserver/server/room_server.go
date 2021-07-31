package server

import (
	"context"
	"quark/gameserver"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type roomServer struct {
	gameserver.UnimplementedRoomServer

	r *room
}

func NewRoomServer() gameserver.RoomServer {
	return &roomServer{}
}

func (r *roomServer) CreateRoom(context.Context, *gameserver.CreateRoomRequest) (*gameserver.CreateRoomResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRoom not implemented")
}
func (r *roomServer) Service(gameserver.Room_ServiceServer) error {
	return status.Errorf(codes.Unimplemented, "method Service not implemented")
}
