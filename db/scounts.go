package db

// Scount depicts the scount object for interaction with scounts datastore.
type Scount struct {
	Sid         string
	Owner       string
	Title       string
	Description string
}

// ScountId is the 'id' type for scount collection. Sid is the primary key
// or object identifier in the database.
type ScountId struct {
	Sid string
}

// ScountUpdater provides fields for updating scounts.
type ScountUpdater struct {
	Owner string
	Title string
}
