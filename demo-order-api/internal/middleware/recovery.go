package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// Recovery catches panics and logs them as structured JSON with a full stack trace.
// This replaces chi's built-in Recoverer so the triage agent can parse the output.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				stack := string(debug.Stack())

				slog.ErrorContext(r.Context(), "panic recovered",
					"request_id", chimw.GetReqID(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"error", fmt.Sprintf("%v", rv),
					"stack_trace", stack,
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
