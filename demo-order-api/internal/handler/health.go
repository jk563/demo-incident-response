package handler

import (
	"encoding/json"
	"net/http"
)

// Health returns a simple health check response for the ALB target group.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
