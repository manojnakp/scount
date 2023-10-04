package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/manojnakp/scount/db"
)

// ScountSchema is the location for `Scount` JSON schema.
const ScountSchema = "/schema/Scount.json"

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
	return new(ScountQuery), nil
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

// ScountPathWare is the middleware to set context key corresponding
// to "sid" path parameter using ScountKey.
func ScountPathWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := chi.URLParam(r, "sid")
		ctx := context.WithValue(r.Context(), ScountKey, sid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Router constructs a new chi.Router for the ScountResource.
func (res ScountResource) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(Authware)
	r.With(QueryParser(ParseScountQuery)).
		Get("/", res.ListScounts)
	r.With(BodyParser[ScountRequest], Validware[ScountRequest]).
		Post("/", res.CreateScount)
	r.Route("/{sid}", func(r chi.Router) {
		r.Use(ScountPathWare)
		r.Get("/", res.GetScount)
		r.With(BodyParser[ScountUpdater], Validware[ScountUpdater]).
			Patch("/", res.UpdateScount)
		r.Delete("/", res.DeleteScount)
	})
	return r
}

// ServeHTTP implements http.Handler on ScountResource.
func (res ScountResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// ListScounts handles GET requests at `/scounts`.
func (res ScountResource) ListScounts(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = r.Context()
		query = ctx.Value(QueryKey).(*ScountQuery)
		page  = query.Paging.Page
		size  = query.Paging.Size
	)
	// database call
	scounts, err := res.DB.Scounts.Find(
		ctx,
		&db.ScountFilter{Sid: query.Sid, Uid: query.Uid, Owner: query.Owner, Title: query.Title},
		&db.Projector{Order: query.Sort, Paging: &db.Paging{Limit: size, Offset: page * size}},
	)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// build response collection
	list := make([]Scount, 0)
	scounts.Iterator(func(scount db.Scount) bool {
		list = append(list, Scount{
			Schema: ScountSchema,
			Id:     scount.Sid,
			Title:  scount.Title,
			Desc:   scount.Description,
			Owner:  scount.Owner,
		})
		return true
	})
	err = scounts.Err()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	links := PagingLinks("/scounts", r.URL.Query(), scounts.Total())
	LinkHeader(w, links)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

// CreateScount handles POST request at `/scounts`.
func (res ScountResource) CreateScount(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = r.Context()
		body  = ctx.Value(BodyKey).(ScountRequest)
		owner = ctx.Value(AuthUserKey).(string)
		sid   = GenerateID()
	)
	// insert into db
	err := res.DB.Scounts.Insert(
		ctx,
		db.Scount{Sid: sid, Owner: owner, Title: body.Title, Description: body.Desc},
	)
	if err != nil {
		log.Println(err)
	}
	// match error
	switch {
	case errors.Is(err, db.ErrInvalidData), errors.Is(err, db.ErrSyntaxPrivilege):
		w.WriteHeader(http.StatusBadRequest)
		return
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// newly created scount resource location
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", path.Join("/scounts", sid))
	w.WriteHeader(http.StatusOK)
	// json response
	_ = json.NewEncoder(w).Encode(ScountResponse{
		Schema:   "/schema/ScountResponse.json",
		ScountId: sid,
	})
}

// GetScount handles GET requests at `/scounts/{sid}`.
func (res ScountResource) GetScount(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		sid = ctx.Value(ScountKey).(string)
	)
	scount, err := res.DB.Scounts.FindOne(ctx, &db.ScountId{Sid: sid})
	switch {
	case errors.Is(err, db.ErrNoRows): // sid not exist
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil: // failed to query the db
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK) // ALL OK
	_ = json.NewEncoder(w).Encode(Scount{
		Schema: ScountSchema,
		Id:     scount.Sid,
		Title:  scount.Title,
		Desc:   scount.Description,
		Owner:  scount.Owner,
	})
}

// UpdateScount handles PATCH request at `/scounts/{sid}`.
func (res ScountResource) UpdateScount(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		sid     = ctx.Value(ScountKey).(string)
		updater = ctx.Value(BodyKey).(ScountUpdater)
	)
	var zero ScountUpdater
	// nothing to update: success
	if updater == zero {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// database call
	err := res.DB.Scounts.UpdateOne(
		ctx,
		&db.ScountId{Sid: sid},
		&db.ScountUpdater{Owner: updater.Owner, Title: updater.Title},
	)
	switch {
	case errors.Is(err, db.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		return
	case errors.Is(err, db.ErrConflict): // conflict
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil: // unknown error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) // ALL OK
}

// DeleteScount handles DELETE request at `/scounts/{sid}`.
func (res ScountResource) DeleteScount(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		sid = ctx.Value(ScountKey).(string)
	)
	err := res.DB.Scounts.DeleteOne(ctx, &db.ScountId{Sid: sid})
	switch {
	case errors.Is(err, db.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		return
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
