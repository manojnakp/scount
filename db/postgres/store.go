package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/manojnakp/scount/db"
	"github.com/manojnakp/scount/db/internal"
	"log"
	"strings"

	"github.com/lib/pq"
)

// NewStore constructs a [db.Store] from a SQL (postgres supported)
// database connection handle.
func NewStore(DB *sql.DB) *db.Store {
	return &db.Store{
		Users:   UserCollection{DB},
		Scounts: ScountCollection{DB},
	}
}

// Open dials an SQL connection using POSTGRES connection uri and
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
	var pqerr *pq.Error
	if errors.As(err, &pqerr) {
		code := pqerr.Code
		switch code.Class() {
		// integrity_constraint_violation
		case "22":
			return internal.Error{
				Pretty: db.ErrInvalidData,
				Err:    err,
			}
		// data_exception
		case "23":
			return internal.Error{
				Pretty: db.ErrConflict,
				Err:    err,
			}
		// syntax_error_or_access_rule_violation
		case "42":
			return internal.Error{
				Pretty: db.ErrSyntaxPrivilege,
				Err:    err,
			}
		}
	}
	return err
}

// Add defines addition behavior inside templates.
func Add(x, y int) int {
	return x + y
}

// JoinSorter defines `join` operation inside templates.
func JoinSorter(cols []db.Sorter, fallback string) string {
	if len(cols) == 0 {
		return fallback
	}
	const DESC string = " DESC"
	const SEP string = ", "
	order := cols[0]
	var b strings.Builder
	b.WriteString(order.Column.String())
	if order.Desc {
		b.WriteString(DESC)
	}
	for _, order = range cols[1:] {
		b.WriteString(SEP)
		b.WriteString(order.Column.String())
		if order.Desc {
			b.WriteString(DESC)
		}
	}
	return b.String()
}

// Tx is a handy callback wrapper for executing transactions.
func Tx[T any](
	ctx context.Context,
	sqldb *sql.DB,
	callback func(*sql.Tx) (T, error),
) (t T, err error) {
	if sqldb == nil {
		err = db.ErrNil
		return
	}
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
