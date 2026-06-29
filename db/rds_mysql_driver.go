package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"database/sql/driver"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-sql-driver/mysql"
)

const (
	AuthMethodAWSIAM           = "aws_iam"
	AuthMethodUsernamePassword = "username_password"
)

type RdsMysqlDriver struct {
	Host         string
	Port         int
	Username     string
	Password     string
	DatabaseName string
	AuthMethod   string
	AwsRegion    string
}

func RegisterTLS() error {
	pem, err := loadAwsRDSCAPem()
	if err != nil {
		return err
	}
	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return errors.New("failed to append AWS RDS CA")
	}
	err = mysql.RegisterTLSConfig("rds", &tls.Config{
		RootCAs: rootCertPool,
	})
	if err != nil {
		return err
	}
	return nil

}

var registerTLSOnce sync.Once
var registerTLSErr error

func RegisterTLSOnce() error {
	registerTLSOnce.Do(func() {
		registerTLSErr = RegisterTLS()
	})
	return registerTLSErr
}

const _pem = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

func loadAwsRDSCAPem() ([]byte, error) {
	resp, err := http.Get(_pem)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
func NewDriver(config *RdsMysqlDriver) driver.Driver {
	drv := &Driver{config: config}
	if config.AuthMethod == AuthMethodAWSIAM {
		if err := RegisterTLSOnce(); err != nil {
			drv.initErr = err
			return drv
		}
		drv.startRotation()
	}

	return drv
}

type Driver struct {
	drv    mysql.MySQLDriver
	config *RdsMysqlDriver
	token  string
	initErr error
}

func (r *Driver) Open(_ string) (driver.Conn, error) {
	if r.initErr != nil {
		return nil, r.initErr
	}

	var dbEndpoint = fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)

	mysqlConfig := &mysql.Config{
		Addr:                    dbEndpoint,
		DBName:                  r.config.DatabaseName,
		Net:                     "tcp",
		AllowCleartextPasswords: true,
		AllowNativePasswords:    true,
		ParseTime:               true,
		User:                    r.config.Username,
	}

	if r.config.AuthMethod == AuthMethodAWSIAM {
		mysqlConfig.Passwd = r.token
		mysqlConfig.TLSConfig = "rds"
	} else if r.config.AuthMethod == AuthMethodUsernamePassword {
		mysqlConfig.Passwd = r.config.Password
	}
	return r.drv.Open(mysqlConfig.FormatDSN())
}

func (r *Driver) startRotation() {
	// build token for the first time
	if err := r.buildTokenWithRetry(5); err != nil {
		fmt.Println("could not build auth token", err)
	}

	// start token rotation
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			if err := r.buildTokenWithRetry(5); err != nil {
				// exit if token cannot be refresh token
				fmt.Println("could not build auth token", err)
			}

			// rotate after 10 minutes (before current token expired)
		}
	}()
}

func (r *Driver) buildTokenWithRetry(numRetries int) error {
	for {
		if err := r.buildToken(); err != nil {
			if numRetries == 0 {
				return err

			}
			numRetries--
			// retry after 10 seconds
			time.Sleep(10 * time.Second)
			continue
		}

		break
	}
	return nil
}

func (r *Driver) buildToken() error {
	defaultConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("could not load default config", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := auth.BuildAuthToken(
		ctx,
		fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
		r.config.AwsRegion,
		r.config.Username,
		defaultConfig.Credentials)
	if err != nil {
		fmt.Println("could not build auth token", err)
		return err
	}

	r.token = token

	return nil
}
