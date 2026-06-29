package db

import (
	"context"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"gorm.io/gorm"
)

var _ lifecycle.Component = (*Client)(nil)

// Client wraps a GORM connection with lifecycle hooks.
type Client struct {
	DB *gorm.DB
}

func NewClient(cfg *DBConfig) (*Client, error) {
	gormDB, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{DB: gormDB}, nil
}

func (c *Client) Name() string { return "mariadb" }

func (c *Client) Start(_ context.Context) error { return nil }

func (c *Client) Stop(_ context.Context) error {
	if c.DB == nil {
		return nil
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
