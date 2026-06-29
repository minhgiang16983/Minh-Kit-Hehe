package aws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	httpkit "github.com/minhgiang16983/Minh-Kit-Hehe/client/http"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kms/types"
	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	kms "github.com/aws/aws-sdk-go-v2/service/kms"
	types2 "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type KMSConfig struct {
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	KeyID           string
	httpClient      *http.Client
}

type KMS struct {
	client *kms.Client
	keyId  string
	tr     trace.Tracer
}

func NewAwsKMS(ctx context.Context, cfg *KMSConfig) (types.KmsProviderInterface, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	httpClient := cfg.httpClient

	if cfg.httpClient == nil {
		newHttpClient, err := httpkit.NewHttpClient(&httpkit.HttpClientConfig{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}

		httpClient = newHttpClient
	}

	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(cfg.Region),
		awsConfig.WithHTTPClient(httpClient),
	}

	if cfg.AccessKeyId != "-" && cfg.SecretAccessKey != "-" {
		opts = append(opts,
			awsConfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     cfg.AccessKeyId,
					SecretAccessKey: cfg.SecretAccessKey,
				}, nil
			})),
		)
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	client := kms.NewFromConfig(awsCfg)

	var tr trace.Tracer
	if tracing.GetGlobalTracingConfig().IsEnableTracing() {
		name := tracing.GetGlobalTracingConfig().ServiceName
		tr = otel.GetTracerProvider().Tracer(name)
	}

	return &KMS{
		client: client,
		keyId:  cfg.KeyID,
		tr:     tr,
	}, nil
}

func (k *KMS) Encrypt(ctx context.Context, plaintext []byte) ([]byte, string, error) {
	_, end := k.startSpan(ctx, "kms.encrypt")
	defer end()
	resp, err := k.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:               aws.String(k.keyId),
		Plaintext:           plaintext,
		EncryptionAlgorithm: types2.EncryptionAlgorithmSpecSymmetricDefault,
	})

	if err != nil {
		return nil, "", err
	}

	return resp.CiphertextBlob, *resp.KeyId, nil
}

func (k *KMS) Decrypt(ctx context.Context, keyId string, wrappedDek []byte) ([]byte, error) {
	_, end := k.startSpan(ctx, "kms.decrypt")
	defer end()
	resp, err := k.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:      wrappedDek,
		KeyId:               aws.String(keyId),
		EncryptionAlgorithm: types2.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, err
	}

	if resp.Plaintext == nil {
		return nil, errors.New("KMS decrypt failed: no plaintext returned")
	}

	return resp.Plaintext, nil
}

func (k *KMS) startSpan(ctx context.Context, name string) (context.Context, func()) {
	if k.tr == nil {
		return ctx, func() {}
	}
	ctx2, span := k.tr.Start(ctx, name)
	return ctx2, func() { span.End() }
}

func (k *KMS) DescribeKey(ctx context.Context, keyId string) (string, error) {
	_, end := k.startSpan(ctx, "kms.describe-key")
	defer end()

	resp, err := k.client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return "", err
	}

	return *resp.KeyMetadata.KeyId, nil
}

func (k *KMS) ReEncrypt(ctx context.Context, sourceKeyId string, wrappedDEK []byte) ([]byte, string, error) {
	_, end := k.startSpan(ctx, "kms.re-encrypt")
	defer end()

	resp, err := k.client.ReEncrypt(ctx, &kms.ReEncryptInput{
		DestinationKeyId:               aws.String(k.keyId),
		DestinationEncryptionAlgorithm: types2.EncryptionAlgorithmSpecSymmetricDefault,
		CiphertextBlob:                 wrappedDEK,
		SourceEncryptionAlgorithm:      types2.EncryptionAlgorithmSpecSymmetricDefault,
		SourceKeyId:                    aws.String(sourceKeyId),
	})
	if err != nil {
		return nil, "", err
	}

	return resp.CiphertextBlob, *resp.KeyId, nil
}
