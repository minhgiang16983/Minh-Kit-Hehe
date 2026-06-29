package types

import "context"

type KmsProviderInterface interface {
	Encrypt(ctx context.Context, plainText []byte) ([]byte, string, error)
	Decrypt(ctx context.Context, keyId string, wrappedDek []byte) ([]byte, error)
	DescribeKey(ctx context.Context, keyId string) (string, error)
	ReEncrypt(ctx context.Context, sourceKeyId string, wrappedDEK []byte) ([]byte, string, error)
}

type KmsProvider string

const (
	AwsKms KmsProvider = "aws-kms"
)

type EncryptedInfo struct {
	Provider   KmsProvider `json:"provider"`
	Ciphertext []byte      `json:"ciphertext"`
	IV         []byte      `json:"iv"`
	Tag        []byte      `json:"tag"`
	WrappedDEK []byte      `json:"wrappedDEK"`
	KeyVersion string      `json:"key_version"`
}
