package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	gameserver.RegisterRoomServer(grpcServer, room.NewRoomServer())

	go func() {
		log.Printf("gRPC service listen at %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	<-sig

	grpcServer.GracefulStop()
	log.Println("gRPC server shutdown")
}
