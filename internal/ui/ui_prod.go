//go:build !dev

package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

func Handler() http.Handler {
	// Strip the "dist" prefix from the embed.FS
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API routes handled by main router, but double check safety
		if strings.HasPrefix(path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Check if file exists in the FS
		// We use a small hack: try to open it. If it fails, serve index.html
		f, err := dist.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			// File exists (e.g. assets/main.js), close it and let fileServer serve it
			defer f.Close()
			// Ensure we don't serve directory listings
			stat, _ := f.Stat()
			if !stat.IsDir() {
				if strings.HasPrefix(path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback to index.html for SPA routing
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
