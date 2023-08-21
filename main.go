package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/manojnakp/scount/api"
	"github.com/manojnakp/scount/db"
	"github.com/manojnakp/scount/db/postgres"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
)

func main() {
	uri := os.Getenv("DB_URI")
	store, err := postgres.Open(uri)
	if err != nil {
		log.Fatal(err)
	}
	r := chi.NewRouter()
	r.Mount("/docs/", DocHandler{}.Router())
	r.Handle("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently))
	r.Handle("/", &Handler{store})
	_ = http.ListenAndServe(":8080", r)
}

type Handler struct {
	DB *db.Store
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := api.GenereateID()
	user := &db.User{
		Uid:      id,
		Email:    "someone.else@example.com",
		Username: "Some One Else",
		Password: "lsjiaw2g5h",
	}
	err := h.DB.Users.Insert(r.Context(), user)
	if err != nil {
		log.Println(err)
		if errors.Is(err, db.ErrConflict) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(id))
}
