package db

import (
	"context"
	"errors"
)

// some common errors.
var (
	ErrNoRows          = errors.New("db: no rows in result")
	ErrConflict        = errors.New("db: database state violation")
	ErrSyntaxPrivilege = errors.New("db: syntax error or insufficient privilege")
	ErrInvalidData     = errors.New("db: invalid data for database operation")
	ErrNil             = errors.New("db: nil pointer")
	ErrInvalidColumn   = errors.New("db: invalid or non-permissible column")
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
		Insert(ctx context.Context, users ...User) error
		DeleteOne(ctx context.Context, id string) error
		Update(context.Context, *UserFilter, *UserUpdater) error
		Find(context.Context, *UserFilter, *Projector) (List[User], error)
		FindOne(ctx context.Context, id string) (User, error)
		FindByEmail(ctx context.Context, email string) (User, error)
	}
}

// Paging provides pagination options for querying the database.
type Paging struct {
	Limit  int
	Offset int
}

// Projector provides projection options for fetching the
// data from a collection.
type Projector struct {
	Sort   string
	Desc   bool
	Paging *Paging
}

// List is a generic list of items.
type List[T any] struct {
	Data  []T
	Total int
}
