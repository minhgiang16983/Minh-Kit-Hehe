package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

func EncryptWithDEK(ctx context.Context, dek, plaintext []byte) (ciphertext, iv, tag []byte, err error) {

	// Block aes
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, nil, nil, err
	}

	iv = make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}

	ct := gcm.Seal(nil, iv, plaintext, nil)
	if len(ct) < gcm.Overhead() {
		return nil, nil, nil, errors.New("ciphertext too short")
	}

	ciphertext = ct[:len(ct)-gcm.Overhead()]
	tag = ct[len(ct)-gcm.Overhead():]
	return ciphertext, iv, tag, nil
}

func DecryptWithDEK(ctx context.Context, dek, ciphertext, iv, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ct := append(ciphertext, tag...)
	return gcm.Open(nil, iv, ct, nil)
}
