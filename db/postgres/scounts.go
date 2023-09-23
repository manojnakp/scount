package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"text/template"

	"github.com/manojnakp/scount/db"
)

// ScountDeleteQuery is a query statement for deleting a single scount by sid.
const ScountDeleteQuery = `
DELETE FROM scounts
WHERE sid = $1;`

// ScountSelectQuery is a query statement for fetching a single scount by sid.
const ScountSelectQuery = `
SELECT sid, owner, title, description
FROM scounts
WHERE sid = $1;`

// ScountUpdateTemplate is a query template for updating scounts from ScountCollection.
var ScountUpdateTemplate = template.Must(template.New("scount-update").
	Funcs(template.FuncMap{"add": Add}).
	Parse(`
UPDATE scounts SET
{{ range $i, $col := . }}
	{{ $col }} = {{ add $i 2 | printf "$%d" }}
{{ end }}
WHERE sid = $1;
`))

// ScountCollection provides a convenient way to interact with `scounts` table.
type ScountCollection struct {
	DB *sql.DB
}

// DeleteOne removes exactly 1 scount from `scounts` collection based on sid.
func (colln ScountCollection) DeleteOne(ctx context.Context, id *db.ScountId) error {
	if id == nil {
		return db.ErrNil
	}
	res, err := colln.DB.ExecContext(ctx, ScountDeleteQuery, id.Sid)
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

// UpdateOne modifies exactly 1 scount from `scounts` collection.
func (colln ScountCollection) UpdateOne(
	ctx context.Context,
	id *db.ScountId,
	setter *db.ScountUpdater,
) error {
	// pointer validity check
	if id == nil {
		return db.ErrNil
	}
	if setter == nil {
		setter = new(db.ScountUpdater)
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

// buildUpdateQuery constructs a scount update query using provided id,
// setter and ScountUpdateTemplate. id and setter are not nil.
func (colln ScountCollection) buildUpdateQuery(
	id *db.ScountId, setter *db.ScountUpdater,
) (string, []any, error) {
	// dollar arguments in the SQL query
	args := []any{id.Sid}
	// cols for template arguments
	cols := make([]string, 0)
	if setter.Owner != "" {
		cols = append(cols, "owner")
		args = append(args, setter.Owner)
	}
	if setter.Title != "" {
		cols = append(cols, "title")
		args = append(args, setter.Title)
	}
	// construct
	buf := new(bytes.Buffer)
	err := ScountUpdateTemplate.Execute(buf, cols)
	if err != nil {
		log.Println("tmpl exec scount-update: ", err)
		return "", nil, err
	}
	return buf.String(), args, nil
}

// FindOne fetches scount from colln by id.
func (colln ScountCollection) FindOne(ctx context.Context, id *db.ScountId) (s db.Scount, err error) {
	if id == nil {
		err = db.ErrNil
		return
	}
	var scount db.Scount
	err = colln.DB.QueryRowContext(ctx, ScountSelectQuery, id.Sid).
		Scan(&scount.Sid, &scount.Owner, &scount.Title, &scount.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	return scount, nil
}

// compile-time assertion
var _ interface {
	DeleteOne(ctx context.Context, id *db.ScountId) error
	UpdateOne(ctx context.Context, id *db.ScountId, setter *db.ScountUpdater) error
	FindOne(ctx context.Context, id *db.ScountId) (db.Scount, error)
} = ScountCollection{}
