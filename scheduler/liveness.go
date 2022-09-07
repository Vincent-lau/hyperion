package scheduler

import (
	"context"
	"errors"

	"google.golang.org/grpc/health/grpc_health_v1"
)

func (sched *Scheduler) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (sched *Scheduler) Watch(in *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return errors.New("not implemented")
}