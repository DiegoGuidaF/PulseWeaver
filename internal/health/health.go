package health

import (
	"encoding/json"
	"net/http"
	"time"
)

type Response struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func Handler(w http.ResponseWriter, _ *http.Request) {
	response := Response{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
