package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"

	"github.com/IBM/sarama"
)

func createTLSConfiguration(cfg *KafkaTLS) (t *tls.Config) {

	if cfg == nil || !cfg.Enabled {
		return nil
	}

	t = &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	if cfg.CAFile != "" && cfg.ServerName != "" && cfg.InsecureSkipVerify {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}
	}
	return t
}

func ApplyKafkaTLS(sc *sarama.Config, cfg *KafkaTLS) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	sc.Net.TLS.Enable = true
	sc.Net.TLS.Config = createTLSConfiguration(cfg)
	return nil
}
