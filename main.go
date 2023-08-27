package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"

	"github.com/manojnakp/scount/api"
	"github.com/manojnakp/scount/db/postgres"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
)

func main() {
	secret, err := base64.StdEncoding.DecodeString(os.Getenv("SECRET"))
	if err == nil && len(secret) != 0 {
		api.SetKey(secret)
	}
	uri := os.Getenv("DB_URI")
	store, err := postgres.Open(uri)
	if err != nil {
		log.Fatal(err)
	}
	r := chi.NewRouter()
	r.Mount("/docs/", DocHandler{}.Router())
	r.Handle("/docs", http.RedirectHandler("/docs/", http.StatusMovedPermanently))
	r.Mount("/auth", api.AuthResource{DB: store}.Router())
	r.HandleFunc("/health", HealthCheck)
	_ = http.ListenAndServe(":8080", r)
}

// HealthCheck is a status check endpoint, whether server is alive.
func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
