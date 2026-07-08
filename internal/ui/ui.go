package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// Handler serves the web UI at / and static assets under /ui/static/
func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/" || path == "/ui" || path == "/ui/":
			data, err := staticFS.ReadFile("static/index.html")
			if err != nil {
				http.Error(w, "ui unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(data)
		case strings.HasPrefix(path, "/ui/static/"):
			r2 := *r
			u := *r.URL
			u.Path = strings.TrimPrefix(path, "/ui/static")
			r2.URL = &u
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, &r2)
		default:
			http.NotFound(w, r)
		}
	})
}
