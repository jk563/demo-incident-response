package handler

import (
	"net/http"
	"strconv"

	"log/slog"

	"github.com/example/demo-incident-response/demo-order-api/internal/store"
)

// Events groups the HTTP handlers for agent observer events.
type Events struct {
	store *store.EventStore
}

// NewEvents creates an Events handler with the given store.
func NewEvents(s *store.EventStore) *Events {
	return &Events{store: s}
}

// Latest handles GET /api/agent-events/latest — returns the current incident ID.
func (h *Events) Latest(w http.ResponseWriter, r *http.Request) {
	incidentID, err := h.store.LatestIncident(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get latest incident", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get latest incident")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"incident_id": incidentID})
}

// Incidents handles GET /api/agent-events/incidents — returns recent incidents for the dropdown.
func (h *Events) Incidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := h.store.ListIncidents(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list incidents", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	if incidents == nil {
		incidents = []store.IncidentSummary{}
	}
	writeJSON(w, http.StatusOK, incidents)
}

// List handles GET /api/agent-events?incident_id=X&after=0 — returns events after a sequence number.
func (h *Events) List(w http.ResponseWriter, r *http.Request) {
	incidentID := r.URL.Query().Get("incident_id")
	if incidentID == "" {
		writeError(w, http.StatusBadRequest, "incident_id is required")
		return
	}

	afterSeq := 0
	if v := r.URL.Query().Get("after"); v != "" {
		var err error
		afterSeq, err = strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "after must be a number")
			return
		}
	}

	events, err := h.store.ListEvents(r.Context(), incidentID, afterSeq)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list events", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	if events == nil {
		events = []store.AgentEvent{}
	}

	writeJSON(w, http.StatusOK, events)
}
