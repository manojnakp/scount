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

// filesystem embedding `./docs/static` contains json files (mostly schema definitions),
// OpenAPI spec and API documentation as `index.html` file.
//
//go:embed docs/static
var filesystem embed.FS

// DocHandler serves documentation resources.
type DocHandler struct{}

// Router gives a chi router for the DocHandler.
func (d DocHandler) Router() chi.Router {
	r := chi.NewRouter()
	// serve json files like 'openapi.json'
	r.Get("/schema/*", func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParamFromCtx(r.Context(), "*")
		d.ServeFile(w, r, path.Join("schema", filename))
	})
	r.Handle("/docs/", http.RedirectHandler("/docs.html", http.StatusMovedPermanently))
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		d.ServeFile(w, r, "openapi.json")
	})
	// serve html docs
	r.Get("/docs.html", func(w http.ResponseWriter, r *http.Request) {
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
	file, err := filesystem.Open(path.Join("docs/static", filename))
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
