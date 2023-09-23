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
SELECT sid, uid,
count(*) OVER () AS total
FROM members
WHERE ($1 OR sid = $2)
AND ($3 OR uid = $4)
ORDER BY {{ join .Order "sid, uid" }}
{{ with .Paging }}
	LIMIT {{ .Limit }}
	OFFSET {{ .Offset }}
{{ end }};
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
func (colln MemberCollection) FindOne(ctx context.Context, id *db.MemberId) (m db.Member, err error) {
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
) (list db.List[db.Member], err error) {
	// pointer validity check
	if filter == nil {
		filter = new(db.MemberFilter)
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
	var members []db.Member
	var total int
	// scan through the rows
	for rows.Next() {
		var m db.Member
		err = rows.Scan(&m.Sid, &m.Uid)
		if err != nil {
			return
		}
		members = append(members, m)
	}
	if err = rows.Err(); err != nil {
		return
	}
	return db.List[db.Member]{Data: members, Total: total}, nil
}

// buildSelectQuery constructs member select query using provided
// filter, projector and MemberSelectTemplate.
func (colln MemberCollection) buildSelectQuery(
	filter *db.MemberFilter, projector *db.Projector,
) (string, []any, error) {
	// TODO: projector.Order[i] NOT IN db.MemberAllowedCols -> db.ErrInvalidColumn
	args := make([]any, 0)
	// WHERE clause
	args = append(args, filter.Sid == "", filter.Sid)
	args = append(args, filter.Uid == "", filter.Uid)
	// construct
	buf := new(bytes.Buffer)
	err := MemberSelectTemplate.Execute(buf, projector)
	if err != nil {
		log.Println("tmpl exec member-select: ", err)
		return "", nil, err
	}
	return buf.String(), args, nil
}

// compile-time assertion
var _ db.Collection[db.Member, db.MemberFilter, db.MemberUpdater, db.MemberId] = MemberCollection{}
