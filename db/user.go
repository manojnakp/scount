package db

// User depicts the user object for interactions with users datastore.
type User struct {
	Uid      string
	Email    string
	Username string
	Password string
}

// UserFilter provides fields for filtering the users.
type UserFilter struct {
	Uid      string
	Email    string
	Username string
	Password string
}

// UserUpdater provides Fields for updating users.
type UserUpdater struct {
	Username string
	Password string
}
