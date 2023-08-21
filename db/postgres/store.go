package postgres

import (
	"context"
	"database/sql"
	"log"

	"github.com/manojnakp/scount/db"

	"github.com/lib/pq"
)

// NewStore constructs a [db.Store] with postgres as database.
func NewStore(DB *sql.DB) *db.Store {
	return &db.Store{
		Users: UserCollection{DB},
	}
}

// Open dials an sql connection using connection uri and
// constructs a [db.Store]. Wraps over NewStore.
func Open(uri string) (*db.Store, error) {
	sqldb, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}
	return NewStore(sqldb), nil
}

// Error is a utility function for error handling.
func Error(err error) error {
	if err == nil {
		return nil
	}
	log.Printf("db error: %#v", err)
	if pqerr, ok := err.(*pq.Error); ok {
		code := pqerr.Code
		switch code.Class() {
		// integrity_constraint_violation
		case pq.ErrorClass("22"):
			return db.Error{
				Pretty: db.ErrInvalidData,
				Err:    err,
			}
		// data_exception
		case pq.ErrorClass("23"):
			return db.Error{
				Pretty: db.ErrConflict,
				Err:    err,
			}
		// syntax_error_or_access_rule_violation
		case pq.ErrorClass("42"):
			return db.Error{
				Pretty: db.ErrSyntaxPrivilege,
				Err:    err,
			}
		}
	}
	return err
}

// Tx is a handy callback wrapper for executing transactions.
func Tx[T any](
	ctx context.Context,
	sqldb *sql.DB,
	callback func(*sql.Tx) (T, error),
) (t T, err error) {
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	// execute callback
	value, err := callback(tx)
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}
	return value, nil
}
