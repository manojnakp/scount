package api

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/manojnakp/scount/db"
)

// Scount describes the scount resource.
// schema is defined at `Scount.json`.
type Scount struct {
	Schema string `json:"$schema,omitempty"`
	Id     string `json:"id"`
	Title  string `json:"title"`
	Desc   string `json:"description"`
	Owner  string `json:"owner"`
}

// ScountQuery describes the url query parameters
// used for filtering the scounts.
// schema is defined at `ScountQuery.json`.
type ScountQuery struct {
	Sid    string
	Uid    string
	Owner  string
	Title  string
	Sort   []db.Sorter
	Paging Paginator
}

// ParseScountQuery parses the query parameters on scount collection resource.
func ParseScountQuery(url.Values) (*ScountQuery, error) {
	return nil, nil
}

// ScountRequest describes new scount creation request.
// schema is defined at `ScountRequest.json`
type ScountRequest struct {
	Title string `json:"title"`
	Desc  string `json:"description"`
}

// Validate implements Validator on ScountRequest.
func (ScountRequest) Validate() error {
	return nil
}

// ScountResponse points to the newly created scount resource.
// schema is defined at `ScountResponse.json`
type ScountResponse struct {
	Schema   string `json:"$schema,omitempty"`
	ScountId string `json:"scount_id"`
}

// ScountUpdater describes scount resource update request.
// schema is defined at `ScountUpdater.json`
type ScountUpdater struct {
	Title string `json:"title,omitempty"`
	Owner string `json:"owner,omitempty"`
}

// Validate implements Validator on ScountUpdater
func (ScountUpdater) Validate() error {
	return nil
}

// ScountResource is the http.Handler for all requests to `/scounts`.
type ScountResource struct {
	DB *db.Store
}

// Router constructs a new chi.Router for the ScountResource.
func (res ScountResource) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(Authware)
	r.With(QueryParser(ParseScountQuery)).
		Get("/", res.ListScounts)
	r.With(BodyParser[ScountRequest], Validware[ScountRequest]).
		Post("/", res.CreateScount)
	r.Get("/{sid}", res.GetScount)
	r.With(BodyParser[ScountUpdater], Validware[ScountUpdater]).
		Patch("/{sid}", res.UpdateScount)
	r.Delete("/{sid}", res.DeleteScount)
	return r
}

// ServeHTTP implements http.Handler on ScountResource.
func (res ScountResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// ListScounts handles GET requests at `/scounts`.
func (res ScountResource) ListScounts(http.ResponseWriter, *http.Request) {}

// CreateScount handles POST request at `/scounts`.
func (res ScountResource) CreateScount(http.ResponseWriter, *http.Request) {}

// GetScount handles GET requests at `/scounts/{sid}`.
func (res ScountResource) GetScount(http.ResponseWriter, *http.Request) {}

// UpdateScount handles PATCH request at `/scounts/{sid}`.
func (res ScountResource) UpdateScount(http.ResponseWriter, *http.Request) {}

// DeleteScount handles DELETE request at `/scounts/{sid}`.
func (res ScountResource) DeleteScount(http.ResponseWriter, *http.Request) {}
