package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/middleware"
	"helpdesk/internal/outbox"
	"helpdesk/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AppEventHandler struct {
	queries db.Querier
}

func NewAppEventHandler(queries db.Querier) *AppEventHandler {
	return &AppEventHandler{
		queries: queries,
	}
}

// ListEvents handles GET /api/v1/app/tickets/{id}/events
// (Executes Query 2 of 2 for ticket detail with keyset pagination)
func (h *AppEventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	var limit int32 = 30
	if l := q.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = int32(val)
		}
	}

	var afterOccurredAt pgtype.Timestamptz
	if curTime := q.Get("after_occurred_at"); curTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, curTime); err == nil {
			afterOccurredAt = types.TimeToTimestamptz(t)
		}
	}

	var afterID pgtype.UUID
	if curID := q.Get("after_id"); curID != "" {
		if u, err := types.StringToUUID(curID); err == nil {
			afterID = u
		}
	}

	visibility := q.Get("visibility") // "" or "public" or "internal"

	// EXECUTES QUERY 2 OF 2
	events, err := h.queries.ListTicketEventsKeyset(r.Context(), db.ListTicketEventsKeysetParams{
		TicketID:        ticketID,
		Visibility:      pgtype.Text{String: visibility, Valid: visibility != ""},
		AfterOccurredAt: afterOccurredAt,
		AfterID:         afterID,
		Limit:           limit + 1,
	})
	if err != nil {
		slog.Error("ListTicketEventsKeyset failed", "error", err)
		http.Error(w, `{"error":"internal_error","message":"failed to fetch events"}`, http.StatusInternalServerError)
		return
	}

	hasNext := len(events) > int(limit)
	if hasNext {
		events = events[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":    events,
		"has_next": hasNext,
	})
}

type CreateEventRequest struct {
	Kind       string   `json:"kind"`       // "outbound_email", "portal_reply", "internal_note"
	Visibility string   `json:"visibility"` // "public", "internal"
	BodyHTML   string   `json:"body_html"`
	BodyText   string   `json:"body_text"`
	Mentions   []string `json:"mentions,omitempty"`
}

// CreateEvent handles POST /api/v1/app/tickets/{id}/events
func (h *AppEventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	nowTz := types.TimeToTimestamptz(now)

	var authorID pgtype.UUID
	if claims != nil && claims.AgentID != nil {
		authorID, _ = types.StringToUUID(*claims.AgentID)
	}

	kind := req.Kind
	if kind == "" {
		kind = "internal_note"
	}
	visibility := req.Visibility
	if visibility == "" {
		if kind == "internal_note" {
			visibility = "internal"
		} else {
			visibility = "public"
		}
	}

	bodyHTML := req.BodyHTML
	if bodyHTML == "" {
		bodyHTML = "<p>" + req.BodyText + "</p>"
	}

	eventID := types.NewUUIDv7()

	event, err := h.queries.CreateTicketEvent(r.Context(), db.CreateTicketEventParams{
		ID:              eventID,
		TicketID:        ticketID,
		Kind:            kind,
		Visibility:      visibility,
		AuthorType:      "agent",
		AuthorID:        authorID,
		BodyHtml:        bodyHTML,
		BodyText:        req.BodyText,
		Metadata:        []byte("{}"),
		EmailReferences: []string{},
		DeliveryStatus:  "delivered",
		OccurredAt:      nowTz,
	})
	if err != nil {
		slog.Error("CreateTicketEvent failed", "error", err, "ticket_id", ticketIDStr)
		http.Error(w, `{"error":"internal_error","message":"failed to create event: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Add mentions if any
	for _, m := range req.Mentions {
		if agentUUID, err := types.StringToUUID(m); err == nil {
			_ = h.queries.CreateMention(r.Context(), db.CreateMentionParams{
				ID:      types.NewUUIDv7(),
				EventID: eventID,
				AgentID: agentUUID,
			})
		}
	}

	// If public reply by agent, mark first_responded_at if not set
	if visibility == "public" {
		_, _ = h.queries.UpdateTicketFields(r.Context(), db.UpdateTicketFieldsParams{
			ID:                 ticketID,
			FirstRespondedAt:   nowTz,
			LastAgentActivityAt: nowTz,
		})
	} else {
		_, _ = h.queries.UpdateTicketFields(r.Context(), db.UpdateTicketFieldsParams{
			ID:                 ticketID,
			LastAgentActivityAt: nowTz,
		})
	}

	// Queue transactional outbox side-effect
	_ = outbox.Publish(r.Context(), h.queries, "ticket_event", eventID, outbox.EventReplyReceived, map[string]any{
		"event_id":   types.UUIDToString(eventID),
		"ticket_id":  ticketIDStr,
		"kind":       kind,
		"visibility": visibility,
		"body_text":  req.BodyText,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(event)
}
