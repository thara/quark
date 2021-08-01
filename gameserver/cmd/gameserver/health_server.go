package main

import (
	"context"
	"quark/health"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type healthServer struct {
	health.UnimplementedHealthServer
}

func (s *healthServer) Check(context.Context, *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{
		Status: health.HealthCheckResponse_SERVING,
	}, nil
}

func (s *healthServer) Watch(*health.HealthCheckRequest, health.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
