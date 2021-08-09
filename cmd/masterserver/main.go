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

	quark_grpc "quark/grpc"
	"quark/masterserver"
	"quark/proto"
)

var addr string
var internalAddr string

func init() {
	flag.StringVar(&addr, "b", "127.0.0.1:20000", "The lobby application binding address for client")
	flag.StringVar(&internalAddr, "i", "127.0.0.1:50000", "The masterserver gRPC binding address for gameserver")
}

func main() {
	flag.Parse()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)

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

	fleet := masterserver.NewFleet()

	// for client
	grpcLobbyServer := grpc.NewServer(opts...)
	{
		proto.RegisterLobbyServer(grpcLobbyServer, quark_grpc.NewLobbyServer(fleet))

		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		go func() {
			log.Printf("Lobby server listen at %s", addr)
			if err := grpcLobbyServer.Serve(lis); err != nil {
				log.Fatal(err)
			}
		}()
	}

	// for gameserver
	grpcMasterServer := grpc.NewServer(opts...)
	{
		proto.RegisterMasterServerServer(grpcMasterServer, quark_grpc.NewMasterServer(fleet))

		lis, err := net.Listen("tcp", internalAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		go func() {
			log.Printf("master server listen at %s", internalAddr)
			if err := grpcMasterServer.Serve(lis); err != nil {
				log.Fatal(err)
			}
		}()
	}

	<-sig

	grpcMasterServer.GracefulStop()
	grpcLobbyServer.GracefulStop()

	log.Println("gRPC server shutdown")
}
