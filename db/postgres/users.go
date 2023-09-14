package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"html/template"
	"log"
	"slices"

	"github.com/manojnakp/scount/db"
)

// UserInsertQuery is query statement for inserting single user.
const UserInsertQuery = `
INSERT INTO users (uid, email, username, password)
VALUES ($1, $2, $3, $4);`

// UserDeleteQuery is a query statement for deleting single user by uid.
const UserDeleteQuery = `
DELETE FROM users WHERE uid = $1;`

// UserSelectQuery is a query statement for fetching single user by uid.
const UserSelectQuery = `
SELECT uid, email, username, password
FROM users
WHERE uid = $1;`

// UserByEmailQuery is a query statement for fetching single user by email.
const UserByEmailQuery = `
SELECT uid, email, username, password
FROM users
WHERE email = $1;`

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

// UserSelectTemplate is a query template for finding users from UserCollection.
var UserSelectTemplate = template.Must(template.New("user-select").
	Parse(`
{{ $sort := "uid" }}
{{ if .Sort }} {{ $sort = .Sort }} {{ end }}
SELECT uid, email, username, password,
count(uid) OVER () AS total
FROM users
WHERE ($1 OR uid = $2)
AND ($3 OR email = $4)
AND ($5 OR username ILIKE $6)
AND ($7 OR password = $8)
ORDER BY {{ $sort }} {{ if .Desc }} DESC {{ end }}
{{ with .Paging }}
	LIMIT {{ .Limit }}
	OFFSET {{ .Offset }}
{{ end }};
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

// buildArgs constructs query arguments using provided filter.
func (colln UserCollection) buildArgs(filter *db.UserFilter) []any {
	if filter == nil {
		filter = new(db.UserFilter)
	}
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Uid == "", filter.Uid)
	args = append(args, filter.Email == "", filter.Email)
	args = append(args, filter.Username == "", filter.Username)
	args = append(args, filter.Password == "", filter.Password)
	return args
}

// buildUpdateQuery constructs user update query using provided filter,
// setter and UserUpdateTemplate.
func (colln UserCollection) buildUpdateQuery(
	filter *db.UserFilter,
	setter *db.UserUpdater,
) (string, []any, error) {
	if setter == nil {
		setter = new(db.UserUpdater)
	}
	args := colln.buildArgs(filter)
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

// FindOne fetches user from colln by id.
func (colln UserCollection) FindOne(ctx context.Context, id string) (u db.User, err error) {
	var user db.User
	err = colln.DB.QueryRowContext(ctx, UserSelectQuery, id).
		Scan(&user.Uid, &user.Email, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	return user, nil
}

// FindByEmail fetches user from colln by email.
func (colln UserCollection) FindByEmail(ctx context.Context, email string) (u db.User, err error) {
	var user db.User
	err = colln.DB.QueryRowContext(ctx, UserByEmailQuery, email).
		Scan(&user.Uid, &user.Email, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	return user, nil
}

// Find fetches all the users from colln subject to filter and projector
// options specified.
func (colln UserCollection) Find(
	ctx context.Context,
	filter *db.UserFilter,
	projector *db.Projector,
) (list db.List[db.User], err error) {
	// construct query from template
	query, args, err := colln.buildSelectQuery(filter, projector)
	if err != nil {
		return
	}
	// execute query
	rows, err := colln.DB.QueryContext(ctx, query, args...)
	if err != nil {
		err = Error(err)
		return
	}
	defer rows.Close()
	var users []db.User
	var total int
	// scan through the rows
	for rows.Next() {
		var u db.User
		err = rows.Scan(&u.Uid, &u.Email, &u.Username, &u.Password, &total)
		if err != nil {
			return
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return
	}
	return db.List[db.User]{Data: users, Total: total}, nil
}

// buildSelectQuery constructs user select query using
// provided filter, projector and UserSelectTemplate.
func (colln UserCollection) buildSelectQuery(
	filter *db.UserFilter,
	projector *db.Projector,
) (string, []any, error) {
	if projector == nil {
		projector = new(db.Projector)
	}
	if !slices.Contains(db.UserAllowedCols, projector.Sort) {
		return "", nil, db.ErrInvalidColumn
	}
	args := colln.buildArgs(filter)
	// construct
	buf := new(bytes.Buffer)
	err := UserSelectTemplate.Execute(buf, projector)
	if err != nil {
		log.Println("tmpl exec user-select: ", err)
		return "", nil, err
	}
	return buf.String(), args, nil
}
