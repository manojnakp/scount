package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log"

	"github.com/manojnakp/scount/db"
	"github.com/manojnakp/scount/db/internal"
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

// UserPasswordQuery is a query statement for updating password of the user.
const UserPasswordQuery = `
UPDATE users
SET password = $2
WHERE uid = $1 AND password = $3;`

// UserUpdateTemplate is a query template for updating users from UserCollection.
var UserUpdateTemplate = template.Must(template.New("user-update").
	Funcs(template.FuncMap{"add": Add}).
	Parse(`
UPDATE users SET
{{ range $i, $col := . }}
	{{ $col }} = {{ add $i 2 | printf "$%d" }}
{{ end }}
WHERE uid = $1;
`))

// UserSelectTemplate is a query template for finding users from UserCollection.
var UserSelectTemplate = template.Must(template.New("user-select").
	Funcs(template.FuncMap{"join": JoinSorter}).
	Parse(`
SELECT uid, email, username, password,
count(*) OVER () AS total
FROM users
WHERE ($1 OR uid = $2)
AND ($3 OR email = $4)
AND ($5 OR username ILIKE $6)
ORDER BY {{ join .Order "uid" }}
{{ with .Paging }}
	LIMIT {{ .Limit }}
	OFFSET {{ .Offset }}
{{ end }};
`))

// UserCollection provides a convenient way to interact with `users` table.
type UserCollection struct {
	DB *sql.DB // underlying database handle
}

// Insert adds one or more users to colln. db.ErrNoRows if no users to insert.
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
			password := base64.StdEncoding.EncodeToString(u.Password)
			res, err := stmt.ExecContext(ctx, u.Uid, u.Email, u.Username, password)
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
func (colln UserCollection) DeleteOne(ctx context.Context, id *db.UserId) error {
	if id == nil {
		return db.ErrNil
	}
	res, err := colln.DB.ExecContext(ctx, UserDeleteQuery, id.Uid)
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

// UpdatePassword modifies the password of the matching user record as specified.
func (colln UserCollection) UpdatePassword(ctx context.Context, updater *db.PasswordUpdater) error {
	if updater == nil {
		return db.ErrNil
	}
	res, err := colln.DB.ExecContext(
		ctx, UserPasswordQuery, updater.Uid,
		base64.StdEncoding.EncodeToString(updater.New),
		base64.StdEncoding.EncodeToString(updater.Old),
	)
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

// UpdateOne modifies exactly 1 user from `users` collection.
func (colln UserCollection) UpdateOne(
	ctx context.Context,
	id *db.UserId,
	setter *db.UserUpdater,
) error {
	// pointer validity check
	if id == nil {
		return db.ErrNil
	}
	if setter == nil {
		setter = new(db.UserUpdater)
	}
	// construct query from template
	query, args, err := colln.buildUpdateQuery(id, setter)
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

// buildUpdateQuery constructs user update query using provided id,
// setter and UserUpdateTemplate. id and setter not nil.
func (colln UserCollection) buildUpdateQuery(
	id *db.UserId,
	setter *db.UserUpdater,
) (string, []any, error) {
	// dollar arguments in the SQL query
	args := []any{id.Uid}
	// cols for template arguments
	cols := make([]string, 0)
	if setter.Username != "" {
		cols = append(cols, "username")
		args = append(args, setter.Username)
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
func (colln UserCollection) FindOne(ctx context.Context, id *db.UserId) (u db.User, err error) {
	if id == nil {
		err = db.ErrNil
		return
	}
	var user db.User
	var password string
	err = colln.DB.QueryRowContext(ctx, UserSelectQuery, id.Uid).
		Scan(&user.Uid, &user.Email, &user.Username, &password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	user.Password, err = base64.StdEncoding.DecodeString(password)
	if err != nil {
		err = internal.Error{Pretty: db.ErrEncoding, Err: err}
		return
	}
	return user, nil
}

// FindByEmail fetches user from colln by email.
func (colln UserCollection) FindByEmail(ctx context.Context, email string) (u db.User, err error) {
	var user db.User
	var password string
	err = colln.DB.QueryRowContext(ctx, UserByEmailQuery, email).
		Scan(&user.Uid, &user.Email, &user.Username, &password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	user.Password, err = base64.StdEncoding.DecodeString(password)
	if err != nil {
		err = internal.Error{Pretty: db.ErrEncoding, Err: err}
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
	// pointer validity check
	if filter == nil {
		filter = new(db.UserFilter)
	}
	if projector == nil {
		projector = new(db.Projector)
	}
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
		var password string
		err = rows.Scan(&u.Uid, &u.Email, &u.Username, &password, &total)
		if err != nil {
			return
		}
		u.Password, err = base64.StdEncoding.DecodeString(password)
		if err != nil {
			err = internal.Error{Pretty: db.ErrEncoding, Err: err}
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
	// TODO: projector.Order[i] NOT IN db.UserAllowedCols -> db.ErrInvalidColumn
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Uid == "", filter.Uid)
	args = append(args, filter.Email == "", filter.Email)
	args = append(args, filter.Username == "", filter.Username)
	// construct
	buf := new(bytes.Buffer)
	err := UserSelectTemplate.Execute(buf, projector)
	if err != nil {
		log.Println("tmpl exec user-select: ", err)
		return "", nil, err
	}
	return buf.String(), args, nil
}

// compile-time assertion
var _ interface {
	db.Collection[db.User, db.UserFilter, db.UserUpdater, db.UserId]
	FindByEmail(ctx context.Context, email string) (db.User, error)
	UpdatePassword(ctx context.Context, updater *db.PasswordUpdater) error
} = UserCollection{}
