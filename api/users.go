package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/manojnakp/scount/api/internal"

	"github.com/manojnakp/scount/db"

	"github.com/go-chi/chi/v5"
)

// UserSchema is the location for `User` JSON schema.
const UserSchema = "/schema/User.json"

// ErrUserQuery defines parsing errors for UserQuery.
var ErrUserQuery = errors.New("invalid user query parameters")

// UserSorter is the default sort order for user queries.
var UserSorter = []db.Sorter{
	{
		Column: "uid",
	},
}

// UserQuery defines the query parameters for user collection resource.
// Schema defined in `UserQuery.json`.
type UserQuery struct {
	Id     string
	Email  string
	Name   string
	Sort   []db.Sorter
	Paging Paginator
}

// ParseUserQuery parses the query parameters on user collection resource.
func ParseUserQuery(query url.Values) (*UserQuery, error) {
	id := query.Get("id")
	email := query.Get("email")
	name := query.Get("name")
	paging, err := ParsePaginator(query)
	if err != nil {
		return nil, err
	}
	sort := strings.Split(query.Get("sort"), ",")
	if len(sort) == 0 {
		// default sort condition
		return &UserQuery{
			Id:     id,
			Email:  email,
			Name:   name,
			Sort:   UserSorter,
			Paging: paging,
		}, nil
	}
	list := make([]db.Sorter, 0)
	for _, s := range sort {
		sorter, ok := internal.UserSortMap[strings.TrimSpace(s)]
		if !ok {
			return nil, fmt.Errorf("%w: invalid 'sort' parameter", ErrUserQuery)
		}
		list = append(list, sorter)
	}
	return &UserQuery{
		Id:     id,
		Email:  email,
		Name:   name,
		Sort:   list,
		Paging: paging,
	}, nil
}

// User is the JSON response body for user request fetch request.
// JSON schema defined in `User.json`.
type User struct {
	Schema string `json:"$schema,omitempty"`
	Id     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// UserUpdater is JSON request for updating (PATCH) `/me` resource.
type UserUpdater struct {
	// /docs/UserUpdater.json
	// Schema string `json:"$schema,omitempty"`
	Username string `json:"username,omitempty"`
}

// Validate implements Validator on UserUpdater.
func (UserUpdater) Validate() error {
	return nil
}

// UserResource is http.Handler for all requests to `/users`.
type UserResource struct {
	DB *db.Store
}

// Router constructs a new chi.Router for the UserResource.
func (res UserResource) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(Authware)
	r.With(QueryParser(ParseUserQuery)).
		Get("/", res.ListUsers)
	r.Get("/{uid}", res.GetUser)
	r.With(res.setCurrentUser).
		Get("/me", res.GetCurrentUser)
	r.Patch("/me", res.UpdateUser)
	r.Delete("/me", res.DeleteUser)
	return r
}

// ServeHTTP implements http.Handler on UserResource.
func (res UserResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// fetch obtains user resource for the given user id (obtained from context).
func (res UserResource) fetch(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(UserKey).(string)
	user, err := res.DB.Users.FindOne(r.Context(), &db.UserId{Uid: id})
	switch {
	case errors.Is(err, db.ErrNoRows): // uid not exist
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil: // failed to query the db
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK) // ALL OK
	_ = json.NewEncoder(w).Encode(User{
		Schema: UserSchema,
		Id:     user.Uid,
		Email:  user.Email,
		Name:   user.Username,
	})
}

// setCurrentUser sets the currently logged-in user id at UserKey.
func (res UserResource) setCurrentUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uid := ctx.Value(AuthUserKey).(string)
		ctx = context.WithValue(ctx, UserKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser handles requests at `/users/{uid}`.
func (res UserResource) GetUser(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")
	ctx := context.WithValue(r.Context(), UserKey, uid)
	res.fetch(w, r.WithContext(ctx))
}

// GetCurrentUser is http.HandlerFunc for `/users/me` GET request.
func (res UserResource) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	res.fetch(w, r)
}

// UpdateUser is http.HandlerFunc for `/users/me` PATCH request.
func (res UserResource) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(UserKey).(string)
	updater := r.Context().Value(BodyKey).(UserUpdater)
	var zero UserUpdater
	// nothing to update: success
	if updater == zero {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// database call
	err := res.DB.Users.UpdateOne(
		r.Context(),
		&db.UserId{Uid: id},
		&db.UserUpdater{Username: updater.Username},
	)
	switch {
	case errors.Is(err, db.ErrNoRows):
		// also considered success
	case errors.Is(err, db.ErrConflict): // conflict
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil: // unknown error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) // ALL OK
}

// DeleteUser handles DELETE method on `users/me` route.
func (res UserResource) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(UserKey).(string)
	err := res.DB.Users.DeleteOne(r.Context(), &db.UserId{Uid: id})
	switch {
	case errors.Is(err, db.ErrNoRows):
	// also considered success
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListUsers handles requests at `/users`.
func (res UserResource) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := ctx.Value(QueryKey).(*UserQuery)
	page, size := query.Paging.Page, query.Paging.Size
	// database call
	users, err := res.DB.Users.Find(
		ctx,
		&db.UserFilter{Uid: query.Id, Email: query.Email, Username: query.Name},
		&db.Projector{
			Order:  query.Sort,
			Paging: &db.Paging{Limit: size, Offset: size * page},
		},
	)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// build response collection
	list := make([]User, 0)
	users.Iterator(func(u db.User) bool {
		list = append(list, User{
			Schema: UserSchema,
			Id:     u.Uid,
			Email:  u.Email,
			Name:   u.Username,
		})
		return true
	})
	err = users.Err()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	links := PagingLinks("/users", r.URL.Query(), users.Total())
	LinkHeader(w, links)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list) // respond
}
