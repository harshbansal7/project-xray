package xray

import (
	"crypto/rand"
	"encoding/hex"
)

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "xray-id-fallback"
	}
	return hex.EncodeToString(b[:])
}
