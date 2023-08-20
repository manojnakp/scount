package postgres

import "database/sql"

// UserCollection provides a convenient way to interact
// with `users` table.
type UserCollection struct {
	DB *sql.DB // underlying database handle
}
