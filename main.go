package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

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

// Handler is a home route http handler. Debugging code goes in this handler.
type Handler struct {
	DB *db.Store
}

// ServeHTTP implements http.Handler on Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := h.DB.Users.FindByEmail(r.Context(), "someone@example.com")
	if err != nil {
		if errors.Is(err, db.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("%v: %#v", err, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(user)
}
