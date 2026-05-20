//go:build prod

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

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// API routes handled by main router, but double check safety
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Check if file exists in the FS; if so, serve it directly.
		// If not (or it's a directory), fall back to index.html for SPA routing.
		f, err := dist.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !stat.IsDir() {
				if strings.HasPrefix(path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback to index.html for SPA routing
		w.Header().Set("Cache-Control", "no-cache")
		r2 := *r
		u2 := *r.URL
		u2.Path = "/"
		r2.URL = &u2
		fileServer.ServeHTTP(w, &r2)
	})
}
