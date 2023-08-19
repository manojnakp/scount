package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
)

//go:embed docs
var filesystem embed.FS

func main() {
	uri := os.Getenv("DB_URI")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Fatal(err)
	}
	r := chi.NewRouter()
	r.Get("/docs/{file}.json", func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParam(r, "file")
		file, err := filesystem.Open(path.Join("docs", filename+".json"))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			log.Print(err)
			return
		}
		log.Print("ok")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, file)
	})
	r.Get("/docs/", func(w http.ResponseWriter, r *http.Request) {
		file, err := filesystem.Open(path.Join("docs", "index.html"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.Copy(w, file)
	})
	r.Handle("/", &Handler{db})
	_ = http.ListenAndServe(":8080", r)
}

type Handler struct {
	DB *sql.DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	var data int
	row := h.DB.QueryRow("SELECT 5 AS data")
	err := row.Scan(&data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	_, _ = fmt.Fprintf(w, "DB Conn successful: data=%d", data)
}
