package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"
	"gorm.io/plugin/prometheus"
)

type DBConfig struct {
	DriverName      string        `mapstructure:"driver_name" yaml:"driver_name"`
	Username        string        `mapstructure:"username" yaml:"username"`
	Password        string        `mapstructure:"password" yaml:"password"`
	Host            string        `mapstructure:"host" yaml:"host"`
	DatabaseName    string        `mapstructure:"database_name" yaml:"database_name"`
	Port            int           `mapstructure:"port" yaml:"port"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	LogLevel        string        `mapstructure:"log_level" yaml:"log_level"`
	AuthMethod      string        `mapstructure:"auth_method" yaml:"auth_method"`
	AwsRegion       string        `mapstructure:"aws_region" yaml:"aws_region"`
}

func New(cfg *DBConfig) (*gorm.DB, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	// Data source name. This common format
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}

	mariaDbConfig := cfg

	if cfg.DriverName == "" {
		mariaDbConfig.DriverName = "mysql"
	}

	if cfg.AuthMethod == "" {
		mariaDbConfig.AuthMethod = AuthMethodUsernamePassword
	}

	var dialector gorm.Dialector

	var err error
	logger := NewLogger(mappingLogLevel(cfg.LogLevel))
	dsn :=
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			mariaDbConfig.Username,
			mariaDbConfig.Password,
			mariaDbConfig.Host,
			mariaDbConfig.Port,
			mariaDbConfig.DatabaseName,
		)

	switch mariaDbConfig.AuthMethod {
	case AuthMethodUsernamePassword:
		dialector = mysql.Open(dsn)
	case AuthMethodAWSIAM:
		if mariaDbConfig.DriverName == "" || mariaDbConfig.DriverName == "mysql" {
			// default to rds driver
			mariaDbConfig.DriverName = "rds"
		}

		drv := NewDriver(&RdsMysqlDriver{
			Host:         mariaDbConfig.Host,
			Port:         mariaDbConfig.Port,
			Username:     mariaDbConfig.Username,
			Password:     mariaDbConfig.Password,
			DatabaseName: mariaDbConfig.DatabaseName,
			AuthMethod:   AuthMethodAWSIAM,
			AwsRegion:    mariaDbConfig.AwsRegion,
		})
		sql.Register(mariaDbConfig.DriverName, drv)
		// Register new RDS MySQL driver
		dialector = mysql.New(mysql.Config{
			DriverName: mariaDbConfig.DriverName,
			DSN:        dsn,
		})
	default:
		return nil, errors.New("invalid auth method")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	err = db.Use(prometheus.New(prometheus.Config{
		DBName:          cfg.DatabaseName,
		RefreshInterval: 15,
		MetricsCollector: []prometheus.MetricsCollector{
			&prometheus.MySQL{
				Prefix: "gorm_status",
			},
		},
	}))
	if err != nil {
		return nil, err
	}
	if tracing.GetGlobalTracingConfig().IsEnableDBTracing() {
		err = db.Use(gormtracing.NewPlugin())
		if err != nil {
			return nil, err
		}
	}

	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDb.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDb.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDb.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDb.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Ping
	if err := db.Raw("SELECT 1").Error; err != nil {
		return nil, err
	}

	return db, nil
}

func mappingLogLevel(logLevel string) logger.LogLevel {
	logLevel = strings.ToUpper(logLevel)
	switch logLevel {
	case "SILENT":
		return logger.Silent
	case "ERROR":
		return logger.Error
	case "WARN":
		return logger.Warn
	case "INFO":
		return logger.Info
	case "DEBUG":
		return logger.Info
	default:
		return logger.Info
	}
}
