package api

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"log"
	"strings"
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

// GenerateToken constructs JWT with given subject and signs it with
// given key. Token is set to an expiry time of 1 hour.
func GenerateToken(subject string, key []byte) ([]byte, error) {
	token := jwt.New()
	_ = token.Set(jwt.SubjectKey, subject)
	_ = token.Set(jwt.ExpirationKey, time.Now().Add(time.Hour))
	payload, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, key))
	if err != nil {
		return nil, err
	}
	return payload, nil
}
