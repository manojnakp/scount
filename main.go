package main

import (
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
	err := h.DB.Users.DeleteOne(r.Context(), "3533ic355kpzoccy")
	if err == nil || errors.Is(err, db.ErrNoRows) {
		// log ErrNoRows??
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if errors.Is(err, db.ErrConflict) {
		log.Println(err)
		w.WriteHeader(http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}
