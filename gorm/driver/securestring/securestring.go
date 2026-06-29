package securestring

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/kms"
	"github.com/minhgiang16983/Minh-Kit-Hehe/kms/types"
)

var kmsClient kms.KmsInterface

func SetKmsClient(client kms.KmsInterface) {
	kmsClient = client
}

type SecureStringDriver struct {
	rawText       string
	encryptedInfo *types.EncryptedInfo
}

func NewSecureStringDriver(rawText string) *SecureStringDriver {
	return &SecureStringDriver{
		rawText: rawText,
	}
}

// Read
func (s *SecureStringDriver) Scan(value interface{}) error {

	if value == nil {
		return nil
	}

	var rawTextStr string
	switch value.(type) {
	case []byte:
		rawTextStr = string(value.([]byte))
	case string:
		rawTextStr = value.(string)
	default:
		return errors.New("encrypt: Scan source is not string or []byte")
	}

	if rawTextStr == "" {
		return nil
	}

	blob, err := base64.StdEncoding.DecodeString(rawTextStr)
	if err != nil {
		return errors.New("encrypt: Scan source is not base64 encoded string")
	}

	var decryptedInfo types.EncryptedInfo
	err = json.Unmarshal(blob, &decryptedInfo)
	if err != nil {
		return errors.New("encrypt: Scan source is not json securestring")
	}

	s.encryptedInfo = &decryptedInfo

	return nil
}

// Write
func (s *SecureStringDriver) Value() (driver.Value, error) {

	if s.rawText == "" {
		return nil, nil
	}

	if s.encryptedInfo == nil {

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		encryptedInfo, err := kmsClient.Encrypt(ctx, []byte(s.rawText))
		if err != nil {
			return nil, err
		}

		s.encryptedInfo = encryptedInfo
	}

	v, err := json.Marshal(s.encryptedInfo)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.EncodeToString(v), nil
}

func (s *SecureStringDriver) String(ctx context.Context) (string, error) {
	if s.encryptedInfo == nil {
		return "", nil
	}

	if s.rawText != "" {
		return s.rawText, nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	plainText, err := kmsClient.Decrypt(ctx, s.encryptedInfo)
	if err != nil {
		return "", err
	}

	s.rawText = string(plainText)

	return s.rawText, nil
}
