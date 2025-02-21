package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateStreamKey() string {
	key := make([]byte, 16) // 128-bit key
	_, err := rand.Read(key)
	if err != nil {
		panic("Failed to generate stream key")
	}
	return hex.EncodeToString(key)
}
