package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"path"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-chi/chi/v5"
	"github.com/manojnakp/scount/db"
)

// BCryptCost is the cost used for bcrypt password hashing.
const BCryptCost = 12

// RegisterRequest is the JSON request body format
// at the `/auth/register` endpoint.
type RegisterRequest struct {
	// /docs/RegisterRequest.schema.json
	// Schema string `json:"$schema,omitempty"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate implements Validator on RegisterRequest.
// Simply check non-empty.
func (r RegisterRequest) Validate() error {
	if r.Username == "" ||
		r.Email == "" ||
		r.Password == "" {
		return errors.New("api: validation failed")
	}
	return nil
}

// RegisterResponse is the JSON response format
// at the `auth/register` endpoint.
type RegisterResponse struct {
	// `/docs/RegisterResponse.schema.json`
	Schema string `json:"$schema,omitempty"`
	UserId string `json:"user_id"`
}

// AuthResource is http.Handler for all requests to `/auth`
type AuthResource struct {
	DB *db.Store
}

// Router constructs a new chi router for the auth resource.
func (res AuthResource) Router() chi.Router {
	r := chi.NewRouter()
	r.With(BodyParser[RegisterRequest], Validware[RegisterRequest]).
		Post("/register", res.register)
	return r
}

// ServeHTTP implements http.Handler on AuthResource.
func (res AuthResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := res.Router()
	mux.ServeHTTP(w, r)
}

// register handles user sign up.
func (res AuthResource) register(w http.ResponseWriter, r *http.Request) {
	body := r.Context().Value(BodyKey).(RegisterRequest)
	uid := GenerateID()
	buf, err := bcrypt.GenerateFromPassword([]byte(body.Password), BCryptCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	password := base64.StdEncoding.EncodeToString(buf)
	// insert into db
	err = res.DB.Users.Insert(r.Context(), db.User{
		Uid:      uid,
		Email:    body.Email,
		Username: body.Username,
		Password: password,
	})
	// match error
	switch {
	case errors.Is(err, db.ErrInvalidData),
		errors.Is(err, db.ErrSyntaxPrivilege):
		w.WriteHeader(http.StatusBadRequest)
		return
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// newly created user resource at location
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", path.Join("/users/", uid))
	w.WriteHeader(http.StatusOK)
	// json response
	_ = json.NewEncoder(w).Encode(RegisterResponse{
		Schema: path.Join("/docs/", "RegisterResponse.schema.json"),
		UserId: uid,
	})
}
