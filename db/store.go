package db

import (
	"context"
	"errors"
)

var (
	ErrNoRows          = errors.New("db: no rows in result")
	ErrConflict        = errors.New("db: database state violation")
	ErrSyntaxPrivilege = errors.New("db: syntax error or insufficient privilege")
	ErrInvalidData     = errors.New("db: invalid data for database operation")
)

// assertion
var _ error = Error{}

type Error struct {
	Pretty error
	Err    error
}

// Error implements [error] on Error.
func (err Error) Error() string {
	return err.Pretty.Error()
}

// Unwrap provides support for errors.Is.
func (err Error) Unwrap() []error {
	return []error{err.Pretty, err.Err}
}

// Store provides an interface for all datastore operations in one place.
type Store struct {
	Users interface {
		Insert(ctx context.Context, users ...*User) error
	}
}
