package stores

import "gorm.io/gorm"

type StoreService struct {
}

func NewStoreService(mariaDB *gorm.DB) *StoreService {
	return &StoreService{}
}
