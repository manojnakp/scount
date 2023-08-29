package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/manojnakp/scount/db"

	"github.com/go-chi/chi/v5"
)

var userAllowedSort = map[string]bool{
	"id": true, "email": true,
	"username": true, "": true,
}

var allowedOrder = map[string]bool{
	"asc": true, "dsc": true, "": true,
}

const PageSize = 5

// UserInfo is the JSON response body for user request fetch request.
type UserInfo struct {
	Id    string
	Email string
	Name  string
}

// UserResource is http.Handler for all requests to `/users`.
type UserResource struct {
	DB *db.Store
}

// Router constructs a new chi.Router for the UserResource.
func (res UserResource) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(Authware)
	r.Get("/", res.list)
	r.Get("/{uid}", res.fetch)
	return r
}

// ServeHTTP implements http.Handler on UserResource.
func (res UserResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// fetch handles requests at `/users/{uid}`.
func (res UserResource) fetch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "uid")                       // url parameter
	user, err := res.DB.Users.FindOne(r.Context(), id) // database call
	switch {
	case errors.Is(err, db.ErrNoRows): // uid not exist
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil: // failed to query the db
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK) // ALL OK
	_ = json.NewEncoder(w).Encode(UserInfo{
		Id:    user.Uid,
		Email: user.Email,
		Name:  user.Username,
	})
}

// list handles requests at `/users`.
func (res UserResource) list(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() // populates url query parameters to r.Form.
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	query := listParams{
		id:    r.Form.Get("id"),
		email: r.Form.Get("email"),
		name:  r.Form.Get("name"),
		sort:  r.Form.Get("sort"),
		order: r.Form.Get("order"),
		size:  r.Form.Get("size"),
		page:  r.Form.Get("page"),
	}
	// validation
	if !userAllowedSort[query.sort] || !allowedOrder[query.order] {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	escaper := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	query.name = "%" + escaper.Replace(query.name) + "%"
	filter := &db.UserFilter{
		Uid:      query.id,
		Email:    query.email,
		Username: query.name,
	}
	size, page := PageSize, 0
	// parse input data
	if query.size != "" {
		size, err = strconv.Atoi(query.size)
	}
	if query.page != "" {
		page, err = strconv.Atoi(query.page)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	projector := &db.Projector{
		Sort: query.sort,
		Desc: query.order == "dsc",
		Paging: &db.Paging{
			Limit:  size,
			Offset: size * page,
		},
	}
	// database call
	users, err := res.DB.Users.Find(r.Context(), filter, projector)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// build response collection
	total := users.Total
	list := make([]UserInfo, 0, len(users.Data))
	for _, x := range users.Data {
		list = append(list, UserInfo{
			Id:    x.Uid,
			Email: x.Email,
			Name:  x.Username,
		})
	}
	// link header
	if total > 0 {
		header := res.linkHeader(r.URL.Query(), page, 0, total/size)
		w.Header().Set("Link", header)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list) // respond
}

func (res UserResource) linkHeader(params url.Values, page, first, last int) string {
	links := make([]string, 0, 4)
	params.Set("page", strconv.Itoa(first))
	links = append(links, linkHeader(params, "first"))
	params.Set("page", strconv.Itoa(last))
	links = append(links, linkHeader(params, "last"))
	if page > 0 {
		params.Set("page", strconv.Itoa(page-1))
		links = append(links, linkHeader(params, "prev"))
	}
	if page < last {
		params.Set("page", strconv.Itoa(page+1))
		links = append(links, linkHeader(params, "next"))
	}
	return strings.Join(links, ", ")
}

// linkHeader returns the header of the form `</users?page=1>; rel="prev"`.
func linkHeader(qs url.Values, rel string) string {
	return fmt.Sprintf("<%s>; rel=%q", "/users?"+qs.Encode(), rel)
}

// listParams is used for parsing the query parameters for list handler.
type listParams struct {
	id, email, name string
	sort, order     string
	size, page      string
}
