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

// ScountInsertQuery is a query statement for adding a single scount by id.
const ScountInsertQuery = `
WITH mcte AS (
	INSERT INTO scounts (sid, owner, title, description)
	VALUES ($1, $2, $3, $4)
	RETURNING sid, owner
)
INSERT INTO members
SELECT sid, owner FROM mcte;
`

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

// ScountSelectTemplate is a query template for finding scounts from ScountCollection.
var ScountSelectTemplate = template.Must(template.New("scount-select").
	Funcs(template.FuncMap{"join": JoinSorter}).
	Parse(`
{{ define "filter" }}
	FROM scounts NATURAL JOIN members
	WHERE ($1 OR sid = $2)
	AND ($3 OR uid = $4)
	AND ($5 OR owner = $6)
	AND ($7 OR title ILIKE $8)
	ORDER BY {{ join .Order "sid, uid" }}
{{ end }}

{{ define "find" }}
	SELECT DISTINCT sid, owner, title, description,
	{{ template "filter" }}
	{{ with .Paging }}
		LIMIT {{ .Limit }}
		OFFSET {{ .Offset }}
	{{ end }};
{{ end }}

{{ define "count" }}
	SELECT count(*) AS total
	{{ template "filter" }};
{{ end }}
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

// Insert adds one or more scounts into colln. db.ErrNoRows if empty scounts.
func (colln ScountCollection) Insert(ctx context.Context, scounts ...db.Scount) error {
	if len(scounts) == 0 {
		return db.ErrNoRows
	}
	_, err := Tx[struct{}](ctx, colln.DB, func(tx *sql.Tx) (struct{}, error) {
		var zero struct{}
		// prepare insert query
		stmt, err := tx.PrepareContext(ctx, ScountInsertQuery)
		if err != nil {
			log.Println("invalid stmt to prepare: ", err)
			return zero, err
		}
		defer stmt.Close()
		// insert every scount
		for _, s := range scounts {
			res, err := stmt.ExecContext(ctx, s.Sid, s.Owner, s.Title, s.Description)
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

// Find fetches all the scounts from colln subject to filter and projector
// options specified.
func (colln ScountCollection) Find(
	ctx context.Context,
	filter *db.ScountFilter,
	projector *db.Projector,
) (list db.List[db.Scount], err error) {
	args := colln.buildArgs(filter)
	counter, finder, err := colln.buildSelectQuery(projector)
	if err != nil {
		return
	}
	iterator := func(yield func(db.Scount) bool) (int, error) {
		return Tx[int](ctx, colln.DB, func(tx *sql.Tx) (int, error) {
			return queryData[db.Scount]{
				context: ctx,
				sqldb:   tx,
				counter: counter,
				finder:  finder,
				args:    args,
				scanner: colln.scanOne,
			}.iterator(yield)
		})
	}
	iterable := db.NewIterable[db.Scount](iterator)
	return db.ListFromIterable(iterable)
}

// scanOne scans one scount from rows and returns associated data.
func (colln ScountCollection) scanOne(rows *sql.Rows) (s db.Scount, err error) {
	var scount db.Scount
	err = rows.Scan(&scount.Sid, &scount.Owner, &scount.Title, &scount.Description)
	if err != nil {
		return
	}
	return scount, nil
}

// buildSelectQuery constructs scount select query using
// provided filter, projector and SCountSelectTemplate.
func (colln ScountCollection) buildSelectQuery(projector *db.Projector) (string, string, error) {
	// TODO: projector.Order[i] NOT IN db.ScountAllowedCols -> db.ErrInvalidColumn
	if projector == nil {
		projector = new(db.Projector)
	}
	// construct count query
	buf := new(bytes.Buffer)
	err := ScountSelectTemplate.ExecuteTemplate(buf, "count", projector)
	if err != nil {
		log.Println("tmpl exec scount-select: ", err)
		return "", "", err
	}
	counter := buf.String()
	// construct find query
	buf.Reset()
	err = ScountSelectTemplate.ExecuteTemplate(buf, "find", projector)
	if err != nil {
		return "", "", err
	}
	finder := buf.String()
	return counter, finder, nil
}

// buildArgs constructs sql dollar argument values for executing the query.
func (colln ScountCollection) buildArgs(filter *db.ScountFilter) []any {
	if filter == nil {
		filter = new(db.ScountFilter)
	}
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Sid == "", filter.Sid)
	args = append(args, filter.Uid == "", filter.Uid)
	args = append(args, filter.Owner == "", filter.Owner)
	args = append(args, filter.Title == "", filter.Title)
	return args
}

// compile-time assertion
var _ db.Collection[db.Scount, db.ScountFilter, db.ScountUpdater, db.ScountId] = ScountCollection{}
