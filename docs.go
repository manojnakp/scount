package main

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
)

// filesystem embedding `./docs` contains json files (mostly schema defintions),
// OpenAPI spec and API documentation as `index.html` file.
//
//go:embed docs
var filesystem embed.FS

// DocHandler serves documentation resources.
type DocHandler struct{}

// Router gives a chi router for the DocHandler.
func (d DocHandler) Router() chi.Router {
	r := chi.NewRouter()
	// serve json files like 'openapi.json'
	r.Get("/{file}", func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParamFromCtx(r.Context(), "file")
		d.ServeFile(w, r, filename)
	})
	r.Handle("/index.html", http.RedirectHandler("/docs/", http.StatusMovedPermanently))
	// serve html docs
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		d.ServeFile(w, r, "index.html")
	})
	return r
}

// ServeHTTP implements http.Handler on DocHandler.
func (d DocHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.Router().ServeHTTP(w, r)
}

// ServeFile is a utility method for streaming a file from `/docs`.
func (d DocHandler) ServeFile(w http.ResponseWriter, _ *http.Request, filename string) {
	file, err := filesystem.Open(path.Join("docs", filename))
	if err != nil {
		log.Println(err)
		// all these known errors to be treated as '404: Not Found'
		if errors.Is(err, fs.ErrInvalid) ||
			errors.Is(err, fs.ErrPermission) ||
			errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// otherwise server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	extension := path.Ext(filename)
	mediatype := mime.TypeByExtension(extension)
	if mediatype != "" {
		w.Header().Set("Content-Type", mediatype)
	}
	_, _ = io.Copy(w, file)
}
