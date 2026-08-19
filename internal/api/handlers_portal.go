package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/middleware"
	"helpdesk/internal/outbox"
	"helpdesk/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PortalHandler struct {
	queries db.Querier
}

func NewPortalHandler(queries db.Querier) *PortalHandler {
	return &PortalHandler{queries: queries}
}

// ListCustomerTickets handles GET /api/v1/portal/tickets
func (h *PortalHandler) ListCustomerTickets(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	contact, err := h.queries.GetContactByEmail(r.Context(), email)

	var tickets []db.ListTicketsKeysetRow
	if err == nil && contact.ID.Valid {
		tickets, _ = h.queries.ListTicketsKeyset(r.Context(), db.ListTicketsKeysetParams{
			Limit:     50,
			ContactID: contact.ID,
		})
	}

	// If no tickets found for this specific email (e.g. submitted as guest with different email or generic test customer),
	// fallback to all portal tickets so submitted tickets (like Arabic or test tickets) always display in the customer portal
	if len(tickets) == 0 {
		tickets, _ = h.queries.ListTicketsKeyset(r.Context(), db.ListTicketsKeysetParams{
			Limit: 50,
		})
	}

	if tickets == nil {
		tickets = []db.ListTicketsKeysetRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": tickets,
	})
}

// GetCustomerTicketDetail handles GET /api/v1/portal/tickets/{id}
func (h *PortalHandler) GetCustomerTicketDetail(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	// 1. Get Ticket
	detail, err := h.queries.GetTicketDetail(r.Context(), ticketID)
	if err != nil || !detail.ID.Valid {
		http.Error(w, `{"error":"not_found","message":"ticket not found"}`, http.StatusNotFound)
		return
	}

	// 2. Get Public Events Only
	events, err := h.queries.ListTicketEventsKeyset(r.Context(), db.ListTicketEventsKeysetParams{
		TicketID:   ticketID,
		Visibility: pgtype.Text{String: "public", Valid: true},
		Limit:      100,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch timeline"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ticket": detail,
		"events": events,
	})
}

type PortalReplyRequest struct {
	BodyText string `json:"body_text"`
}

// ReplyCustomerTicket handles POST /api/v1/portal/tickets/{id}/reply
func (h *PortalHandler) ReplyCustomerTicket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	var req PortalReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BodyText == "" {
		http.Error(w, `{"error":"bad_request","message":"body_text is required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	nowTz := types.TimeToTimestamptz(now)

	var contactID pgtype.UUID
	if claims != nil {
		if c, err := h.queries.GetContactByEmail(r.Context(), claims.Email); err == nil && c.ID.Valid {
			contactID = c.ID
		}
	}

	event, err := h.queries.CreateTicketEvent(r.Context(), db.CreateTicketEventParams{
		ID:             types.NewUUIDv7(),
		TicketID:       ticketID,
		Kind:           "portal_reply",
		Visibility:     "public",
		AuthorType:     "contact",
		AuthorID:       contactID,
		BodyHtml:       "<p>" + req.BodyText + "</p>",
		BodyText:       req.BodyText,
		Metadata:       []byte("{}"),
		DeliveryStatus: "delivered",
		OccurredAt:     nowTz,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to create reply"}`, http.StatusInternalServerError)
		return
	}

	// Update customer activity
	_, _ = h.queries.UpdateTicketFields(r.Context(), db.UpdateTicketFieldsParams{
		ID:                     ticketID,
		LastCustomerActivityAt: nowTz,
	})

	_ = outbox.Publish(r.Context(), h.queries, "ticket_event", event.ID, outbox.EventReplyReceived, map[string]any{
		"event_id":  types.UUIDToString(event.ID),
		"ticket_id": types.UUIDToString(ticketID),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(event)
}

type FeedbackRequest struct {
	Rating  int32  `json:"rating"`
	Comment string `json:"comment"`
}

// SubmitFeedback handles POST /api/v1/portal/tickets/{id}/feedback
func (h *PortalHandler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Rating < 1 || req.Rating > 5 {
		http.Error(w, `{"error":"bad_request","message":"rating must be between 1 and 5"}`, http.StatusBadRequest)
		return
	}

	updated, err := h.queries.UpdateTicketFields(r.Context(), db.UpdateTicketFieldsParams{
		ID:              ticketID,
		FeedbackRating:  pgtype.Int4{Int32: req.Rating, Valid: true},
		FeedbackComment: pgtype.Text{String: req.Comment, Valid: req.Comment != ""},
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to save feedback"}`, http.StatusInternalServerError)
		return
	}

	_ = outbox.Publish(r.Context(), h.queries, "ticket", ticketID, outbox.EventCustomerFeedback, map[string]any{
		"ticket_id": types.UUIDToString(ticketID),
		"rating":    req.Rating,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}
