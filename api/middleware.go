package api

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"net/url"

	"github.com/lestrrat-go/jwx/v2/jwa"

	"github.com/manojnakp/scount/api/internal"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Context keys used for passing data across middlewares.
var (
	BodyKey     internal.BodyKey
	AuthUserKey internal.AuthUserKey
	UserKey     internal.UserKey
	QueryKey    internal.QueryKey
)

// Middleware is a convenient alias for http middleware.
type Middleware = func(http.Handler) http.Handler

// Validator is any type that has validation logic defined on it.
type Validator interface {
	// Validate checks if `this` is valid.
	// nil error upon success and
	// non-nil error means failure.
	Validate() error
}

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

// QueryParser is a generic url query parser. Parsed query parameters
// are stored in context with key QueryKey.
func QueryParser[T any](parser func(url.Values) (T, error)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			query, err := parser(r.Form)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx := context.WithValue(r.Context(), QueryKey, query)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Validware is middleware for validation of request body.
//
// Pre-requisite: BodyKey should be present in request context
// and of type T.
func Validware[T Validator](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// assume BodyKey in context is not `nil` and conforms to `T` type
		body := r.Context().Value(BodyKey).(T)
		err := body.Validate()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Authware is the middleware for handling user authentication
// by validation of auth token in `Authorization` header.
func Authware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := jwt.ParseHeader(
			r.Header, "Authorization",
			jwt.WithContext(r.Context()),
			jwt.WithValidate(true),
			jwt.WithKey(jwa.HS256, GetKey()),
		)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), AuthUserKey, token.Subject())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
