package internal

import (
	"strconv"

	"github.com/manojnakp/scount/db"
)

// UserSortMap maps allowed values of *sort* query parameter to
// corresponding sorters for user resource queries.
var UserSortMap = map[string]db.Sorter{
	"id": {
		Column: "uid",
	},
	"email": {
		Column: "email",
	},
	"name": {
		Column: "username",
	},
	"~id": {
		Column: "uid",
		Desc:   true,
	},
	"~email": {
		Column: "email",
		Desc:   true,
	},
	"~name": {
		Column: "username",
		Desc:   true,
	},
}

// ParseInt is wrapper on strconv.Atoi with default value in case of empty string.
func ParseInt(s string, _default int) (int, error) {
	if s == "" {
		return _default, nil
	}
	return strconv.Atoi(s)
}
