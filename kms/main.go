package kms

import (
	"context"
	"fmt"

	awsKms "github.com/minhgiang16983/Minh-Kit-Hehe/kms/aws"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kms/types"
	utils2 "github.com/minhgiang16983/Minh-Kit-Hehe/kms/utils"
	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"github.com/minhgiang16983/Minh-Kit-Hehe/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type KmsInterface interface {
	Encrypt(ctx context.Context, plaintext []byte) (*types.EncryptedInfo, error)
	Decrypt(ctx context.Context, encryptedInfo *types.EncryptedInfo) ([]byte, error)
	DescribeKey(ctx context.Context, keyId string) (string, error)
	ReEncrypt(ctx context.Context, oldData *types.EncryptedInfo) (*types.EncryptedInfo, error)
}

type Kms struct {
	client   types.KmsProviderInterface
	tr       trace.Tracer
	provider types.KmsProvider
}

type Config struct {
	KmsProvider  types.KmsProvider
	AwsKmsConfig *awsKms.KMSConfig
}

func New(ctx context.Context, cfg *Config) (KmsInterface, error) {
	var tr trace.Tracer = nil
	if tracing.GetGlobalTracingConfig().IsEnableTracing() {
		name := tracing.GetGlobalTracingConfig().ServiceName
		tr = otel.GetTracerProvider().Tracer(name)
	}

	switch cfg.KmsProvider {
	case types.AwsKms:
		client, err := awsKms.NewAwsKMS(ctx, cfg.AwsKmsConfig)
		if err != nil {
			return nil, err
		}

		return &Kms{
			client:   client,
			tr:       tr,
			provider: types.AwsKms,
		}, nil
	default:
		return nil, fmt.Errorf("not supported kms %s provider", cfg.KmsProvider)
	}
}

func (k *Kms) Encrypt(ctx context.Context, plainText []byte) (*types.EncryptedInfo, error) {
	_, end := k.startSpan(ctx, "encrypt")
	defer end()
	dek, err := k.genDek(ctx)
	if err != nil {
		return nil, err
	}

	cipherText, iv, tag, err := k.encryptDek(ctx, dek, plainText)
	if err != nil {
		return nil, err
	}

	wrappedDek, keyVersion, err := k.client.Encrypt(ctx, dek)
	if err != nil {
		return nil, err
	}

	return &types.EncryptedInfo{
		Provider:   k.provider,
		Ciphertext: cipherText,
		IV:         iv,
		Tag:        tag,
		WrappedDEK: wrappedDek,
		KeyVersion: keyVersion,
	}, nil
}

// Generate Data Encryption Key
func (k *Kms) genDek(ctx context.Context) ([]byte, error) {
	_, end := k.startSpan(ctx, "generate.data.encryption.key")
	defer end()
	dek, err := utils.RandomString(32)
	if err != nil {
		return nil, err
	}

	return dek, nil
}

func (k *Kms) encryptDek(ctx context.Context, dek, plainText []byte) ([]byte, []byte, []byte, error) {
	_, end := k.startSpan(ctx, "encrypt.dek")
	defer end()
	cipherText, iv, tag, err := utils2.EncryptWithDEK(ctx, dek, plainText)
	if err != nil {
		return nil, nil, nil, err
	}

	return cipherText, iv, tag, nil
}

func (k *Kms) startSpan(ctx context.Context, name string) (context.Context, func()) {
	if k.tr == nil {
		return ctx, func() {}
	}
	ctx2, span := k.tr.Start(ctx, name)
	return ctx2, func() { span.End() }
}

func (k *Kms) Decrypt(ctx context.Context, encryptedInfo *types.EncryptedInfo) ([]byte, error) {
	_, end := k.startSpan(ctx, "decrypt")
	defer end()

	dek, err := k.client.Decrypt(ctx, encryptedInfo.KeyVersion, encryptedInfo.WrappedDEK)
	if err != nil {
		return nil, err
	}

	return utils2.DecryptWithDEK(ctx, dek, encryptedInfo.Ciphertext, encryptedInfo.IV, encryptedInfo.Tag)
}

func (k *Kms) DescribeKey(ctx context.Context, keyId string) (string, error) {
	_, end := k.startSpan(ctx, "describe-key")
	defer end()
	key, err := k.client.DescribeKey(ctx, keyId)
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func (k *Kms) ReEncrypt(ctx context.Context, oldData *types.EncryptedInfo) (*types.EncryptedInfo, error) {
	_, end := k.startSpan(ctx, "re-encrypt")
	defer end()

	dek, keyVersion, err := k.client.ReEncrypt(ctx, oldData.KeyVersion, oldData.WrappedDEK)
	if err != nil {
		return nil, err
	}

	return &types.EncryptedInfo{
		Provider:   k.provider,
		Ciphertext: oldData.Ciphertext,
		IV:         oldData.IV,
		Tag:        oldData.Tag,
		KeyVersion: keyVersion,
		WrappedDEK: dek,
	}, nil
}
