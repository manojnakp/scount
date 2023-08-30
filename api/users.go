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

// userAllowedSort is the set of allowed values for *sort* query parameter for user resource queries.
var userAllowedSort = map[string]bool{
	"id": true, "email": true,
	"username": true, "": true,
}

// allowedOrder is the set of allowed sort order values.
var allowedOrder = map[string]bool{
	"asc": true, "dsc": true, "": true,
}

// PageSize is the default number of items limit to a page.
const PageSize = 5

// UserInfo is the JSON response body for user request fetch request.
type UserInfo struct {
	// /docs/UserResponse.json
	Schema string `json:"$schema,omitempty"`
	Id     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
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
		Schema: "/docs/UserResponse.json",
		Id:     user.Uid,
		Email:  user.Email,
		Name:   user.Username,
	})
}

// list handles requests at `/users`.
func (res UserResource) list(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() // populates url query parameters to r.Form.
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	query, err := ParseUserParams(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	filter := &db.UserFilter{
		Uid:      query.Id,
		Email:    query.Email,
		Username: query.Name,
	}
	projector := &db.Projector{
		Sort: query.Sort,
		Desc: query.Order == "dsc",
		Paging: &db.Paging{
			Limit:  query.Size,
			Offset: query.Size * query.Page,
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
		header := res.linkHeader(r.URL.Query(), query.Size, 0, total/query.Size)
		w.Header().Set("Link", header)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list) // respond
}

// linkHeader returns `Link` header with pagination details for given
// current, first and last pages.
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

// UserParams is used for parsing the query parameters for list handler.
type UserParams struct {
	Id, Email, Name string
	Sort, Order     string
	Size, Page      int
}

// ParseUserParams parses and validates the given query parameters
// into UserParams that can be consumed by UserResource.list.
func ParseUserParams(qs url.Values) (params UserParams, err error) {
	// validate `name`
	name := qs.Get("name")
	escaper := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	name = "%" + escaper.Replace(name) + "%"
	// validate `sort` and `order`
	sort := qs.Get("sort")
	order := qs.Get("order")
	if !userAllowedSort[sort] || !allowedOrder[order] {
		err = errors.New("api: invalid query parameter")
		return
	}
	size, page := PageSize, 0
	// parse `size` and `page`
	qsize := qs.Get("size")
	qpage := qs.Get("page")
	if qsize != "" {
		size, err = strconv.Atoi(qsize)
	}
	if qpage != "" {
		page, err = strconv.Atoi(qpage)
	}
	if err != nil {
		return
	}
	if size <= 0 || size > 50 {
		err = errors.New("api: invalid query parameter")
		return
	}
	if page < 0 {
		err = errors.New("api: invalid query parameter")
		return
	}
	return UserParams{
		Id:    qs.Get("id"),
		Email: qs.Get("email"),
		Name:  name,
		Sort:  sort,
		Order: order,
		Size:  size,
		Page:  page,
	}, nil
}
