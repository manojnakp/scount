package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	uri := os.Getenv("DB_URI")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Fatal(err)
	}
	_ = http.ListenAndServe(":8080", &Handler{db})
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
