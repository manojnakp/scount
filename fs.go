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
// OpenAPI spec and API documentation as `ref.html` file.
//
//go:embed openapi.json schema doc/ref.html
var filesystem embed.FS

// FileServer serves documentation resources.
type FileServer struct{}

// Router gives a chi router for the FileServer.
func (f FileServer) Router() chi.Router {
	r := chi.NewRouter()
	// serve json files like 'openapi.json'
	r.Get("/schema/*", func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParamFromCtx(r.Context(), "*")
		f.ServeFile(w, path.Join("schema", filename))
	})
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		f.ServeFile(w, "openapi.json")
	})
	// serve html API reference file
	r.Get("/ref.html", func(w http.ResponseWriter, r *http.Request) {
		f.ServeFile(w, "doc/ref.html")
	})
	return r
}

// ServeHTTP implements http.Handler on DocHandler.
func (f FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.Router().ServeHTTP(w, r)
}

// ServeFile is a utility method for streaming a file from `/docs`.
func (FileServer) ServeFile(w http.ResponseWriter, filename string) {
	file, err := filesystem.Open(filename)
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
	mediaType := mime.TypeByExtension(extension)
	if mediaType != "" {
		w.Header().Set("Content-Type", mediaType)
	}
	_, _ = io.Copy(w, file)
}
