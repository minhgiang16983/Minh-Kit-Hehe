package services

import (
	"context"

	pb "github.com/minhgiang16983/service-kit/pb"
)

func (s *ServiceKitService) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status: "OK",
	}, nil
}
