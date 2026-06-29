package baseservice

import (
	"context"

	hskbasepb "github.com/minhgiang16983/Minh-Kit-Hehe/base_service/pb/base"
)

type BaseServiceInterface interface {
	hskbasepb.HskBaseServer
}

type BaseService struct {
	hskbasepb.UnsafeHskBaseServer
}

func New() BaseServiceInterface {
	return &BaseService{}
}

func (s *BaseService) Health(_ context.Context, _ *hskbasepb.HealthRequest) (*hskbasepb.HealthResponse, error) {
	return &hskbasepb.HealthResponse{
		Status: "OK",
	}, nil
}
