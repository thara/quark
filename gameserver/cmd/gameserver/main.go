package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	"quark/gameserver"
	"quark/gameserver/room"
)

var addr string

func init() {
	flag.StringVar(&addr, "b", "127.0.0.1:20000", "The gameserver gRPC binding address")
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	gameserver.RegisterRoomServer(grpcServer, room.NewRoomServer())
	grpcServer.Serve(lis)
}
