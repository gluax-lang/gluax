package common

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
