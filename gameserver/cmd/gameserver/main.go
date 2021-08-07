package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"quark/gameserver/server"
	"quark/proto"
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

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLogger),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(),
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zapLogger),
		)),
	}
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterHealthServer(grpcServer, new(healthServer))
	proto.RegisterRoomServer(grpcServer, server.NewRoomServer())

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
