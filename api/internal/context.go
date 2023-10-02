package internal

// BodyKey is a context key type for (parsed) HTTP request body.
type BodyKey struct{}

// AuthUserKey is a context key type for "sub" of JWT.
type AuthUserKey struct{}

// UserKey is a context key type for `userid` path parameter.
type UserKey struct{}

// QueryKey is a context key type for query parameters
type QueryKey struct{}
