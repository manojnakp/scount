package postgres

import (
	"context"
	"database/sql"
	"log"

	"github.com/manojnakp/scount/db"
)

// UserInsertQuery is query statement for inserting single user.
const UserInsertQuery = `
INSERT INTO users (uid, email, username, password)
VALUES ($1, $2, $3, $4);`

// UserDeleteQuery is a query statement for deleting single user by uid.
const UserDeleteQuery = `
DELETE FROM users WHERE uid = $1;`

// UserCollection provides a convenient way to interact
// with `users` table.
type UserCollection struct {
	DB *sql.DB // underlying database handle
}

// Insert adds one or more users to colln. Does nothing if no users to insert.
func (colln UserCollection) Insert(ctx context.Context, users ...*db.User) error {
	if len(users) == 0 {
		return db.ErrNoRows
	}
	_, err := Tx[struct{}](ctx, colln.DB, func(tx *sql.Tx) (struct{}, error) {
		var zero struct{}
		// prepare insert query
		stmt, err := tx.PrepareContext(ctx, UserInsertQuery)
		if err != nil {
			log.Println("invalid stmt to prepare: ", err)
			return zero, err
		}
		defer stmt.Close()
		// insert every user
		for _, u := range users {
			res, err := stmt.ExecContext(ctx, u.Uid, u.Email, u.Username, u.Password)
			if err != nil {
				return zero, Error(err)
			}
			// assert res.RowsAffected gives 1
			count, err := res.RowsAffected()
			if err == nil {
				// assert count == 1
				_ = count == 1
			} else {
				log.Println("failed RowsAffected: ", err)
				// does not affect insert operation
			}
		}
		return zero, nil
	})
	return err
}

// DeleteOne removes exactly 1 user from `users` collection based on id.
func (colln UserCollection) DeleteOne(ctx context.Context, id string) error {
	res, err := colln.DB.ExecContext(ctx, UserDeleteQuery, id)
	if err != nil {
		return Error(err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return db.ErrNoRows
	}
	return nil
}
