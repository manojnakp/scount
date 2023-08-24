package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"time"
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

// AuthToken is structured format of the auth token.
type AuthToken struct {
	Id     string    `json:"id"`
	Expiry time.Time `json:"exp"`
}

// Generate generates a sufficiently strong crypto random id,
// along with time information of token expiry.
func (token *AuthToken) Generate() error {
	buf := make([]byte, 24)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return err
	}
	token.Id = base64.StdEncoding.
		WithPadding(base64.NoPadding).
		EncodeToString(buf)
	token.Expiry = time.Now().Add(time.Hour)
	return nil
}

// Opaque encodes the auth token into on-the-wire format that
// is opaque to the client and more compact as well as encrypted.
func (token *AuthToken) Opaque(key []byte) (string, error) {
	plaintext, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	// generate nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	// encrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	// encode to opaque string token
	encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	c64 := encoder.EncodeToString(ciphertext)
	n64 := encoder.EncodeToString(nonce)
	return c64 + "." + n64, nil
}

// Parse constructs structured auth token from opaque on-the-wire
// encrypted token.
func (token *AuthToken) Parse(s string, key []byte) (err error) {
	decoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	// s = <ciphertext:base64>.<nonce:base64>
	c64, n64, found := strings.Cut(s, ".")
	if !found {
		err = errors.New("api: invalid auth token format")
		return
	}
	nonce, err := decoder.DecodeString(n64)
	if err != nil {
		return
	}
	ciphertext, err := decoder.DecodeString(c64)
	if err != nil {
		return
	}
	// construct AES block cipher (key ~ aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	// decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return
	}
	// decode
	var t AuthToken
	err = json.Unmarshal(plaintext, &t)
	if err != nil {
		return
	}
	*token = t
	return nil
}
