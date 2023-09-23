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
	ErrEncoding        = errors.New("db: invalid data encoding")
	ErrInvalidColumn   = errors.New("db: invalid or non-permissible column")
)

// Store provides an interface for all datastore operations in one place.
type Store struct {
	Users interface {
		Collection[User, UserFilter, UserUpdater, UserId]
		FindByEmail(ctx context.Context, email string) (User, error)
		UpdatePassword(context.Context, *PasswordUpdater) error
	}
	Scounts Collection[Scount, ScountFilter, ScountUpdater, ScountId]
	Members Collection[Member, MemberFilter, MemberUpdater, MemberId]
}

// Collection is a generic implementation of a collection with
// support for basic CRUD operations. It is assumed to be thread-safe.
type Collection[Item, Filter, Updater, Id any] interface {
	// Insert adds multiple items to the database. If no items, then ErrNoRows.
	Insert(ctx context.Context, items ...Item) error
	// DeleteOne removes exactly one record from the database.
	// If not found, then ErrNoRows. If nil id passed, then ErrNil.
	DeleteOne(ctx context.Context, id *Id) error
	// UpdateOne modifies exactly one record in the database.
	// If no records match then ErrNoRows.
	UpdateOne(context.Context, *Id, *Updater) error
	// Find fetches all the records that match the given filter and projects
	// them as a list. If no records match, then empty list.
	Find(context.Context, *Filter, *Projector) (List[Item], error)
	// FindOne fetches exactly one matching record. If no such record exist
	// in the database, then ErrNoRows.
	FindOne(ctx context.Context, id *Id) (Item, error)
}

// Paging provides pagination options for querying the database.
type Paging struct {
	Limit  int
	Offset int
}

// Projector provides projection options for fetching the
// data from a collection.
type Projector struct {
	Order  []Sorter // column order matters
	Paging *Paging  // pagination options
}

// Sorter defines the sorting order for a particular column.
type Sorter struct {
	Column Column // on which column
	Desc   bool   // descending order
}

// List is a generic list of items.
// TODO: convert into iterator.
type List[T any] struct {
	Data  []T
	Total int
}

// Column defines the column names over which sorting and filtering can
// be performed for a collection.
type Column string

// String implements fmt.Stringer on Column.
func (c Column) String() string {
	return string(c)
}
