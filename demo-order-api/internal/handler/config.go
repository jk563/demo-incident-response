package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

var (
	configOnce sync.Once
	configBody []byte
)

// Config returns observer configuration derived from environment variables.
func Config(w http.ResponseWriter, r *http.Request) {
	configOnce.Do(func() {
		resp := map[string]string{
			"region": os.Getenv("AWS_REGION"),
			"repo":   os.Getenv("GIT_REPO"),
		}
		configBody, _ = json.Marshal(resp)
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(configBody)
}
