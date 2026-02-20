//go:build !prod

// Stub file so that backend can be run without frontend (dist)

package ui

import (
	"net/http"
)

func Handler() http.Handler {
	// In dev mode, we assume you are running the frontend separately (Vite)
	// So we return a simple 404 or a reverse proxy if you wanted to get fancy.
	// But usually, you just don't hit the backend for UI in dev mode.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Optional: Proxy to localhost:5173 if you want "One Port" dev experience
		http.Error(w, "Frontend not embedded. Run 'npm run dev' in frontend/ folder.", http.StatusNotFound)
	})
}
