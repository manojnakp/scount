package api

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

// GenerateID generates cryptographically secure id
// for entities involved in the application. (10 byte base32)
func GenerateID() string {
	buf := make([]byte, 10)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		log.Println(err)
		return ""
	}
	b32 := base32.StdEncoding.
		WithPadding(base32.NoPadding).
		EncodeToString(buf)
	return strings.ToLower(b32)
}

func init() {
	// set random crypto key
	buf := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, buf)
	SetKey(buf)
}

// GenerateToken constructs JWT with given subject and signs it with
// given key. Token is set to an expiry time of 1 hour.
func GenerateToken(subject string) ([]byte, error) {
	token := jwt.New()
	_ = token.Set(jwt.SubjectKey, subject)
	_ = token.Set(jwt.ExpirationKey, time.Now().Add(time.Hour))
	payload, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, GetKey()))
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// secret is private key used for signing JWT.
var secret []byte

// muSecret is the mutex guarding access to secret.
var muSecret sync.Mutex

// SetKey sets secret key used for JWT signing and verification.
// Ownership of `key` parameter is passed, so the client may not modify
// the byte slice.
func SetKey(key []byte) {
	muSecret.Lock()
	defer muSecret.Unlock()
	secret = key
}

// GetKey obtains the secret key used for JWT signing and verify.
// The returned byte slice is supposed to be read-only, may not be
// modified in any way.
func GetKey() []byte {
	muSecret.Lock()
	defer muSecret.Unlock()
	return secret
}
