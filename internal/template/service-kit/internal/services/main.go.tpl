package services

import (
	"github.com/minhgiang16983/service-kit/infra"
	"github.com/minhgiang16983/service-kit/internal/stores"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	pb "github.com/minhgiang16983/service-kit/pb"
)

type ServiceKitService struct {
	pb.UnimplementedServiceKitServer
	Infrastructure infra.InfraInterface
	Store          *stores.StoreService
	l              logger.LoggerInterface
}

func New(
	infrastructure infra.InfraInterface,
	l logger.LoggerInterface,
) *ServiceKitService {

	store := stores.NewStoreService(infrastructure.GetMariaDb())

	return &ServiceKitService{
		Infrastructure: infrastructure,
		Store:          store,
		l:              l,
	}
}
