package db

// Member depicts the member object for interactions with the members datastore.
type Member struct {
	Sid string
	Uid string
}

// MemberId is the 'id' type for member collection. Both sid and uid determine
// a member uniquely.
type MemberId Member

// MemberFilter provides fields for filtering the members.
type MemberFilter Member

// MemberUpdater provides fields necessary for update operation for
// member record.
type MemberUpdater struct{}

// MemberAllowedCols is a list of columns allowed for sorting.
var MemberAllowedCols = []Column{"sid", "uid"}
