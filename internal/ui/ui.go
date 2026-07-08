package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

var spaRoutes = map[string]bool{
	"/":          true,
	"/ui":        true,
	"/ui/":       true,
	"/ui/kv":     true,
	"/ui/tokens": true,
	"/ui/roles":  true,
	"/ui/audit":  true,
	"/ui/watch":  true,
}

func isSPA(path string) bool {
	if spaRoutes[path] {
		return true
	}
	return path == "/ui/kv" || strings.HasPrefix(path, "/ui/kv/")
}

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
		case strings.HasPrefix(path, "/ui/static/"):
			r2 := *r
			u := *r.URL
			u.Path = strings.TrimPrefix(path, "/ui/static")
			r2.URL = &u
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, &r2)
		case isSPA(path):
			data, err := staticFS.ReadFile("static/index.html")
			if err != nil {
				http.Error(w, "ui unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(data)
		default:
			http.NotFound(w, r)
		}
	})
}
