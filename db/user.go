package db

// User depicts the user object for interactions with users datastore.
type User struct {
	Uid      string // id
	Email    string // unique
	Username string
	Password string
}

// UserId is the 'Id' type for user collection. Uid is primary key or
// object identifier in the database.
type UserId struct {
	Uid string
}

// UserFilter provides fields for filtering the users.
type UserFilter struct {
	Uid      string
	Email    string
	Username string
}

// PasswordUpdater provides fields necessary for update
// password operation for a user record.
type PasswordUpdater struct {
	Old string
	New string
	Uid string
}

// UserUpdater provides Fields for updating users.
type UserUpdater struct {
	Username string
}

// UserAllowedCols is a list of columns allowed for sorting.
// Empty string stands for default (uid in this case).
var UserAllowedCols = []Column{"", "uid", "email", "username"}
