package api

import (
	"encoding/json"
	"errors"
	"log"
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
	// /docs/RegisterRequest.json
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
	// `/docs/RegisterResponse.json`
	Schema string `json:"$schema,omitempty"`
	UserId string `json:"user_id"`
}

// LoginRequest is the JSON request body format
// at the `/auth/login` endpoint.
type LoginRequest struct {
	// /docs/LoginRequest.json
	// Schema string `json:"$schema,omitempty"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate implements Validator on LoginRequest.
// Simply check non-empty.
func (r LoginRequest) Validate() error {
	if r.Email == "" || r.Password == "" {
		return errors.New("api: validation failed")
	}
	return nil
}

// LoginResponse is the JSON response body format
// at the `/auth/login` endpoint.
type LoginResponse struct {
	// `/docs/LoginResponse.json`
	Schema string `json:"$schema,omitempty"`
	Token  string `json:"token"`
}

// PasswordChange is the JSON request body format
// at the `/auth/change` endpoint.
type PasswordChange struct {
	// /docs/PasswordChange.json
	// Schema string `json:"$schema,omitempty"`
	Old string `json:"old"`
	New string `json:"new"`
}

// Validate implements Validator on PasswordChange.
// Simply check non-empty.
func (r PasswordChange) Validate() error {
	if r.Old == "" || r.New == "" {
		return errors.New("api: validation failed")
	}
	return nil
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
	r.With(BodyParser[LoginRequest], Validware[LoginRequest]).
		Post("/login", res.login)
	r.With(BodyParser[PasswordChange], Validware[PasswordChange], Authware).
		Post("/change", res.change)
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
	password, err := bcrypt.GenerateFromPassword([]byte(body.Password), BCryptCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// insert into db
	err = res.DB.Users.Insert(r.Context(), db.User{
		Uid:      uid,
		Email:    body.Email,
		Username: body.Username,
		Password: password,
	})
	if err != nil {
		log.Println(err)
	}
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
		Schema: path.Join("/docs/", "RegisterResponse.json"),
		UserId: uid,
	})
}

// login handles user sign in.
func (res AuthResource) login(w http.ResponseWriter, r *http.Request) {
	body := r.Context().Value(BodyKey).(LoginRequest)
	user, err := res.DB.Users.FindByEmail(r.Context(), body.Email)
	if err != nil {
		log.Println(err)
	}
	switch {
	case errors.Is(err, db.ErrNoRows): // not found
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	case err != nil: // db error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword(user.Password, []byte(body.Password))
	if err != nil {
		log.Println(err)
	}
	switch {
	// invalid password provided by client in request body
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	case err != nil: // server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	token, err := GenerateToken(user.Uid)
	if err != nil { // base64 failed to decode
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// json response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Schema: path.Join("/docs/", "LoginResponse.json"),
		Token:  string(token),
	})
}

// change handles password update requests.
func (res AuthResource) change(w http.ResponseWriter, r *http.Request) {
	body := r.Context().Value(BodyKey).(PasswordChange) // BodyParser
	id := r.Context().Value(AuthUserKey).(string)       // Authware
	// fetch user details
	user, err := res.DB.Users.FindOne(r.Context(), &db.UserId{Uid: id})
	if err != nil {
		log.Println(err)
	}
	switch {
	case errors.Is(err, db.ErrNoRows): // no matching user
		w.Header().Set("WWW-Authenticate", `bearer error="invalid_user"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// check if old password is valid
	err = bcrypt.CompareHashAndPassword(user.Password, []byte(body.Old))
	if err != nil {
		log.Println(err)
	}
	switch {
	// old password does not match
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// ok, now generate hash for new password
	password, err := bcrypt.GenerateFromPassword([]byte(body.New), BCryptCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// update database where `old_password = old, id = uid`
	err = res.DB.Users.UpdatePassword(
		r.Context(),
		&db.PasswordUpdater{
			Old: user.Password,
			New: password,
			Uid: user.Uid,
		},
	)
	if err != nil {
		log.Println(err)
	}
	switch {
	case errors.Is(err, db.ErrNoRows): // someone changed db in b/w
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	case errors.Is(err, db.ErrConflict): // update causes a conflict
		w.WriteHeader(http.StatusConflict)
		return
	case err != nil: // some server error happened
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) // ALL OK
	return
}
