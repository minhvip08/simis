package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// GenerateReplicationID generates a random 20-byte replication ID and returns it as an uppercase hexadecimal string.
func GenerateReplicationID() string {
	byte := make([]byte, 20)
	if _, err := rand.Read(byte); err != nil {
		panic(err)
	}
	return strings.ToUpper(hex.EncodeToString(byte))
}
