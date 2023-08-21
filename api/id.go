package api

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

func GenereateID() string {
	buf := make([]byte, 10)
	_, _ = rand.Reader.Read(buf)
	b32 := base32.StdEncoding.
		WithPadding(base32.NoPadding).
		EncodeToString(buf)
	return strings.ToLower(b32)
}
