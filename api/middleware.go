package api

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/manojnakp/scount/api/internal"
)

// BearerPrefix is prefix used in `Authorization` header.
const BearerPrefix = "Bearer "

// Context keys used for passing data across middlewares.
var (
	BodyKey  internal.BodyKey
	TokenKey internal.TokenKey
)

// Middleware is a convenient alias for http middleware.
type Middleware = func(http.Handler) http.Handler

// BodyParser is a generic json body parser. Parsed body is stored in
// context with key BodyKey.
func BodyParser[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body T
		mediatype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		// missing or non-json media type
		if err != nil || mediatype != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			log.Println(err)
			return
		}
		err = json.NewDecoder(r.Body).Decode(&body)
		// request body not conforming to schema
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}
		ctx := context.WithValue(r.Context(), BodyKey, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authware is the middleware for handling user authentication
// by validation of auth token in `Authorization` header.
func Authware(secret [aes.BlockSize]byte) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			prlen := len(BearerPrefix)
			// missing or non bearer authorization header
			if len(header) < prlen ||
				!strings.EqualFold(header[:prlen], BearerPrefix) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				w.WriteHeader(http.StatusUnauthorized)
				log.Println("Authorization != Bearer: ", header)
				return
			}
			opaque := header[prlen:]
			token := new(AuthToken)
			err := token.Parse(opaque, secret[:])
			// token not valid
			if err != nil || token.Id == "" {
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
				w.WriteHeader(http.StatusUnauthorized)
				log.Println(err)
				return
			}
			if token.Expiry.Before(time.Now()) {
				w.Header().Set("WWW-Authenticate", `Bearer error="expired_token"`)
				w.WriteHeader(http.StatusUnauthorized)
				log.Println(token.Expiry, time.Now())
				return
			}
			ctx := context.WithValue(r.Context(), TokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
