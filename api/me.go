package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/manojnakp/scount/db"
)

// MyselfUpdater is JSON request for updating (PATCH) `/me` resource.
type MyselfUpdater struct {
	// /docs/UpdateUser.json
	// Schema string `json:"$schema,omitempty"`
	Username string `json:"username,omitempty"`
}

// Validate implements Validator on MyselfUpdater.
func (MyselfUpdater) Validate() error {
	return nil
}

// MyselfReplacer is JSON request body for replacing (PUT) `/me` resource.
type MyselfReplacer struct {
	// /docs/MyselfReplace.json
	// Schema string `json:"schema,omitempty"`
	Username string `json:"username"`
}

// Validate implements Validator on MyselfReplacer.
func (m MyselfReplacer) Validate() error {
	if m.Username == "" {
		return errors.New("api: validation failed")
	}
	return nil
}

// MyselfResource is http.Handler for all requests to `/me`.
type MyselfResource struct {
	DB *db.Store
}

// Router constructs a new chi.Router for the MyselfResource.
func (res MyselfResource) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(Authware)
	r.Get("/", res.fetch)
	r.Delete("/", res.delete)
	r.With(BodyParser[MyselfUpdater], Validware[MyselfUpdater]).
		Patch("/", res.update)
	r.With(BodyParser[MyselfReplacer], Validware[MyselfReplacer]).
		Put("/", res.replace)
	return r
}

// ServeHTTP implements http.Handler on MyselfResource.
func (res MyselfResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// fetch handles GET method on `/me` route.
func (res MyselfResource) fetch(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(AuthUserKey).(string)
	user, err := res.DB.Users.FindOne(r.Context(), &db.UserId{Uid: id})
	switch {
	case errors.Is(err, db.ErrNoRows):
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UserInfo{
		Schema: "/docs/User.json",
		Id:     user.Uid,
		Email:  user.Email,
		Name:   user.Username,
	})
}

// delete handles DELETE method on `/me` route.
func (res MyselfResource) delete(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(AuthUserKey).(string)
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

// update is http.Handler for PATCH on `/me`.
func (res MyselfResource) update(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(AuthUserKey).(string)
	updater := r.Context().Value(BodyKey).(MyselfUpdater)
	var zero MyselfUpdater
	// nothing to update: success
	if updater == zero {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// database call
	err := res.DB.Users.UpdateOne(r.Context(), &db.UserId{Uid: id}, &db.UserUpdater{
		Username: updater.Username,
	})
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

// replace is http.Handler for `/me` PUT request.
func (res MyselfResource) replace(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(AuthUserKey).(string)
	replacer := r.Context().Value(BodyKey).(MyselfReplacer)
	updater := &db.UserUpdater{Username: replacer.Username}
	// database call
	err := res.DB.Users.UpdateOne(r.Context(), &db.UserId{Uid: id}, updater)
	switch {
	case errors.Is(err, db.ErrNoRows): // nothing to update
		// also considered as success
	case errors.Is(err, db.ErrConflict): // constraint conflict
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil: // unknown error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) // ALL OK
}
