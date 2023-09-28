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

// MemberInsertQuery is query statement for inserting single member.
const MemberInsertQuery = `
INSERT INTO members (sid, uid)
VALUES ($1, $2);`

// MemberDeleteQuery is a query statement for deleting single member by id.
const MemberDeleteQuery = `
DELETE FROM members
WHERE sid = $1 AND uid = $2;`

// MemberSelectQuery is a query statement for fetching single member by id.
const MemberSelectQuery = `
SELECT sid, uid FROM members
WHERE sid = $1 AND uid = $2;`

// NO UPDATE ALLOWED

// MemberSelectTemplate is a query template for finding matching members
// from MemberCollection.
var MemberSelectTemplate = template.Must(template.New("member-select").
	Funcs(template.FuncMap{"join": JoinSorter}).
	Parse(`
{{ define "filter" }}
	FROM members
	WHERE ($1 OR sid = $2)
	AND ($3 OR uid = $4)
	ORDER BY {{ join .Order "sid, uid" }}
{{ end }}

{{ define "find" }}
	SELECT sid, uid,
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

// MemberCollection provides a convenient way to interact with `members` table.
type MemberCollection struct {
	DB *sql.DB
}

// Insert adds one or more members to colln. db.ErrNoRows if no users to insert.
func (colln MemberCollection) Insert(ctx context.Context, members ...db.Member) error {
	if len(members) == 0 {
		return db.ErrNoRows
	}
	_, err := Tx[struct{}](ctx, colln.DB, func(tx *sql.Tx) (struct{}, error) {
		var zero struct{}
		// prepare insert query
		stmt, err := tx.PrepareContext(ctx, MemberInsertQuery)
		if err != nil {
			log.Println("invalid stmt to prepare ", err)
			return zero, err
		}
		defer stmt.Close()
		// insert every member
		for _, m := range members {
			res, err := stmt.ExecContext(ctx, m.Sid, m.Uid)
			if err != nil {
				return zero, Error(err)
			}
			// assert res.RowsAffected gives 1
			count, err := res.RowsAffected()
			if err == nil {
				// assert count == 1
				_ = count
			} else {
				log.Println("failed RowsAffected: ", err)
				// does not affect insert operation
			}
		}
		return zero, nil
	})
	return err
}

// DeleteOne removes exactly 1 member from `members` collection based on matching id.
func (colln MemberCollection) DeleteOne(ctx context.Context, id *db.MemberId) error {
	if id == nil {
		return db.ErrNil
	}
	res, err := colln.DB.ExecContext(ctx, MemberDeleteQuery, id.Sid, id.Uid)
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

// UpdateOne is not supported on `members` collection.
func (colln MemberCollection) UpdateOne(context.Context, *db.MemberId, *db.MemberUpdater) error {
	return errors.ErrUnsupported
}

// FindOne fetches member from colln by id.
func (colln MemberCollection) FindOne(
	ctx context.Context,
	id *db.MemberId,
) (m db.Member, err error) {
	if id == nil {
		err = db.ErrNil
		return
	}
	var member db.Member
	err = colln.DB.QueryRowContext(ctx, MemberSelectQuery, id.Sid, id.Uid).
		Scan(&member.Sid, &member.Uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNoRows
		}
		return
	}
	return member, nil
}

// Find fetches all the members from colln subject to filter and
// projection options specified.
func (colln MemberCollection) Find(
	ctx context.Context,
	filter *db.MemberFilter,
	projector *db.Projector,
) (list *db.Iterable[db.Member], err error) {
	args := colln.buildArgs(filter)
	counter, finder, err := colln.buildSelectQuery(projector)
	if err != nil {
		return
	}
	iterable := func(yield func(db.Member) bool) (int, error) {
		return Tx[int](ctx, colln.DB, func(tx *sql.Tx) (int, error) {
			return queryData[db.Member]{
				context: ctx,
				sqldb:   tx,
				counter: counter,
				finder:  finder,
				args:    args,
				scanner: colln.scanOne,
			}.iterator(yield)
		})
	}
	return db.NewIterable(iterable), nil
}

// scanOne scans one member from rows and returns associated data.
func (colln MemberCollection) scanOne(rows *sql.Rows) (m db.Member, err error) {
	var member db.Member
	err = rows.Scan(&member.Sid, &member.Uid)
	if err != nil {
		return
	}
	return member, nil
}

// buildSelectQuery constructs member select query using provided
// filter, projector and MemberSelectTemplate.
func (colln MemberCollection) buildSelectQuery(projector *db.Projector) (string, string, error) {
	// TODO: projector.Order[i] NOT IN db.MemberAllowedCols -> db.ErrInvalidColumn
	// construct count template
	buf := new(bytes.Buffer)
	err := MemberSelectTemplate.ExecuteTemplate(buf, "count", projector)
	if err != nil {
		return "", "", err
	}
	counter := buf.String()
	// construct find template
	buf.Reset()
	err = MemberSelectTemplate.ExecuteTemplate(buf, "find", projector)
	if err != nil {
		return "", "", err
	}
	finder := buf.String()
	return counter, finder, nil
}

func (colln MemberCollection) buildArgs(filter *db.MemberFilter) []any {
	if filter == nil {
		filter = new(db.MemberFilter)
	}
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Sid == "", filter.Sid)
	args = append(args, filter.Uid == "", filter.Uid)
	return args
}

// compile-time assertion
var _ db.Collection[db.Member, db.MemberFilter, db.MemberUpdater, db.MemberId] = MemberCollection{}
