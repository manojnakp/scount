package main

import (
	"encoding/json"
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
	users, err := h.DB.Users.Find(r.Context(), nil, &db.Projector{
		Paging: &db.Paging{Limit: 1, Offset: 1},
	})
	if err != nil {
		log.Printf("%v: %#v", err, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}
