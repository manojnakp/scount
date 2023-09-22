package internal

import _ "errors"

// assertion
var _ error = Error{}

// Error is a wrapper for error types with a pretty message
// and an underlying error.
type Error struct {
	Pretty error
	Err    error
}

// Error implements [error] on Error.
func (err Error) Error() string {
	return err.Pretty.Error()
}

// Unwrap provides support for [errors.Is].
func (err Error) Unwrap() []error {
	return []error{err.Pretty, err.Err}
}
