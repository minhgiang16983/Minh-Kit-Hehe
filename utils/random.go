package utils

import "crypto/rand"

func RandomString(length int) ([]byte, error) {
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
