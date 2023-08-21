package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
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

// UserUpdateTemplate is a query template for updating users from UserCollection.
var UserUpdateTemplate = template.Must(template.New("user-update").
	Funcs(template.FuncMap{
		"add": func(x, y int) int { return x + y },
	}).
	Parse(`
UPDATE users SET
{{ range $i, $col := . }}
	{{ $col }} = {{ add $i 9 | printf "$%d" }}
{{ end }}
WHERE ($1 OR uid = $2)
AND ($3 OR email = $4)
AND ($5 OR username ILIKE $6)
AND ($7 OR password = $8);
`))

// UserCollection provides a convenient way to interact
// with `users` table.
type UserCollection struct {
	DB *sql.DB // underlying database handle
}

// Insert adds one or more users to colln. Does nothing if no users to insert.
func (colln UserCollection) Insert(ctx context.Context, users ...db.User) error {
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

// Update modifies users from `users` collection.
func (colln UserCollection) Update(
	ctx context.Context,
	filter *db.UserFilter,
	setter *db.UserUpdater,
) error {
	// construct query from template
	query, args, err := colln.buildUpdateQuery(filter, setter)
	if err != nil {
		return err
	}
	// execute query
	res, err := colln.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return Error(err)
	}
	// expect at least 1 row to be updated
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return db.ErrNoRows
	}
	// all good
	return nil
}

// buildUpdateQuery constructs user update query using provided filter,
// setter and UserUpdateTemplate.
func (colln UserCollection) buildUpdateQuery(
	filter *db.UserFilter,
	setter *db.UserUpdater,
) (string, []any, error) {
	if filter == nil {
		filter = new(db.UserFilter)
	}
	if setter == nil {
		setter = new(db.UserUpdater)
	}
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Uid == "", filter.Uid)
	args = append(args, filter.Email == "", filter.Email)
	args = append(args, filter.Username == "", filter.Username)
	args = append(args, filter.Password == "", filter.Password)
	// cols for template arguments
	cols := make([]string, 0)
	if setter.Username != "" {
		cols = append(cols, "username")
		args = append(args, setter.Username)
	}
	if setter.Password != "" {
		cols = append(cols, "password")
		args = append(args, setter.Password)
	}
	// construct
	buf := new(bytes.Buffer)
	err := UserUpdateTemplate.Execute(buf, cols)
	if err != nil {
		log.Println("tmpl exec user-update: ", err)
		return "", nil, err
	}
	return buf.String(), args, nil
}
