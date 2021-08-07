package main

import (
	"context"
	"quark/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type healthServer struct {
	proto.UnimplementedHealthServer
}

func (s *healthServer) Check(context.Context, *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{
		Status: proto.HealthCheckResponse_SERVING,
	}, nil
}

func (s *healthServer) Watch(*proto.HealthCheckRequest, proto.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
