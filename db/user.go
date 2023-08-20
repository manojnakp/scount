package db

// User depicts the user object for interactions with users datastore.
type User struct {
	Uid      string
	Email    string
	Username string
	Password string
}
