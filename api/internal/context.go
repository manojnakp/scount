package internal

// Context key types.
type (
	// BodyKey is context key type for (parsed) HTTP request body.
	BodyKey struct{}
	// AuthUserKey is context key type for "sub" of JWT.
	AuthUserKey struct{}
	// UserKey is context key type for `userid` path parameter.
	UserKey struct{}
	// QueryKey is context key type for query parameters.
	QueryKey struct{}
	// ScountKey is context key type for `sid` path parameter.
	ScountKey struct{}
)
