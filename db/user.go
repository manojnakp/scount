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

// UserAllowedCols is a list of columns allowed for sorting.
// Empty string stands for default (uid in this case).
var UserAllowedCols = []string{"", "uid", "email", "username"}
