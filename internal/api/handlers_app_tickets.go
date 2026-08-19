package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/middleware"
	"helpdesk/internal/outbox"
	"helpdesk/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AppTicketHandler struct {
	queries db.Querier
}

func NewAppTicketHandler(queries db.Querier) *AppTicketHandler {
	return &AppTicketHandler{
		queries: queries,
	}
}

type KeysetResponse struct {
	Items      any     `json:"items"`
	HasNext    bool    `json:"has_next"`
	NextCursor *Cursor `json:"next_cursor,omitempty"`
}

type Cursor struct {
	AfterUpdatedAt string `json:"after_updated_at"`
	AfterID        string `json:"after_id"`
}

// ListTickets handles GET /api/v1/app/tickets (STRICTLY 1 QUERY)
func (h *AppTicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := int32(25)
	if l := q.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = int32(val)
		}
	}

	var afterUpdatedAt pgtype.Timestamptz
	var afterID pgtype.UUID

	if curTime := q.Get("after_updated_at"); curTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, curTime); err == nil {
			afterUpdatedAt = types.TimeToTimestamptz(t)
		}
	}
	if curID := q.Get("after_id"); curID != "" {
		if u, err := types.StringToUUID(curID); err == nil {
			afterID = u
		}
	}

	var statusCategory pgtype.Text
	if sc := strings.ToLower(strings.TrimSpace(q.Get("status_category"))); sc != "" && sc != "all" {
		statusCategory = pgtype.Text{String: sc, Valid: true}
	}

	var agentID pgtype.UUID
	if ag := q.Get("assigned_agent_id"); ag != "" {
		if u, err := types.StringToUUID(ag); err == nil {
			agentID = u
		}
	}

	var teamID pgtype.UUID
	if tm := q.Get("assigned_team_id"); tm != "" {
		if u, err := types.StringToUUID(tm); err == nil {
			teamID = u
		}
	}

	var searchQuery pgtype.Text
	if s := q.Get("search"); s != "" {
		searchQuery = pgtype.Text{String: s, Valid: true}
	}

	// EXECUTES EXACTLY 1 QUERY
	items, err := h.queries.ListTicketsKeyset(r.Context(), db.ListTicketsKeysetParams{
		Limit:           limit + 1, // Request limit+1 to determine has_next without extra COUNT query
		StatusCategory:  statusCategory,
		AssignedAgentID: agentID,
		AssignedTeamID:  teamID,
		SearchQuery:     searchQuery,
		AfterUpdatedAt:  afterUpdatedAt,
		AfterID:         afterID,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch tickets"}`, http.StatusInternalServerError)
		return
	}

	hasNext := len(items) > int(limit)
	if hasNext {
		items = items[:limit]
	}

	var nextCursor *Cursor
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = &Cursor{
			AfterUpdatedAt: types.TimestamptzToNullTime(last.UpdatedAt).Format(time.RFC3339Nano),
			AfterID:        types.UUIDToString(last.ID),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(KeysetResponse{
		Items:      items,
		HasNext:    hasNext,
		NextCursor: nextCursor,
	})
}

// GetTicketDetail handles GET /api/v1/app/tickets/{id} (QUERY 1 of 2)
func (h *AppTicketHandler) GetTicketDetail(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	// EXECUTES QUERY 1 OF 2
	detail, err := h.queries.GetTicketDetail(r.Context(), ticketID)
	if err != nil || !detail.ID.Valid {
		http.Error(w, `{"error":"not_found","message":"ticket not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

type CreateTicketRequest struct {
	Subject         string   `json:"subject"`
	Description     string   `json:"description"`
	ContactEmail    string   `json:"contact_email"`
	ContactName     string   `json:"contact_name"`
	PriorityID      string   `json:"priority_id"`
	TypeID          *string  `json:"type_id,omitempty"`
	AssignedTeamID  *string  `json:"assigned_team_id,omitempty"`
	AssignedAgentID *string  `json:"assigned_agent_id,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

// CreateTicket handles POST /api/v1/app/tickets
func (h *AppTicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Subject == "" || req.Description == "" || req.ContactEmail == "" {
		http.Error(w, `{"error":"bad_request","message":"subject, description, and contact_email are required"}`, http.StatusBadRequest)
		return
	}

	// Resolve contact
	contact, err := h.queries.GetContactByEmail(r.Context(), req.ContactEmail)
	if err != nil || !contact.ID.Valid {
		cName := req.ContactName
		if cName == "" {
			cName = req.ContactEmail
		}
		contact, err = h.queries.CreateContact(r.Context(), db.CreateContactParams{
			ID:           types.NewUUIDv7(),
			PrimaryEmail: req.ContactEmail,
			FullName:     cName,
			Locale:       "en",
			Timezone:     "UTC",
			IsVerified:   true,
			CustomData:   []byte("{}"),
		})
		if err != nil {
			http.Error(w, `{"error":"internal_error","message":"failed to create contact"}`, http.StatusInternalServerError)
			return
		}
	}

	ticketID := types.NewUUIDv7()

	var priorityUUID pgtype.UUID
	if req.PriorityID != "" {
		priorityUUID, _ = types.StringToUUID(req.PriorityID)
	} else {
		priorityUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x02, 0x02}, Valid: true} // Medium
	}

	var typeUUID pgtype.UUID
	if req.TypeID != nil && *req.TypeID != "" {
		typeUUID, _ = types.StringToUUID(*req.TypeID)
	}

	var teamUUID pgtype.UUID
	if req.AssignedTeamID != nil && *req.AssignedTeamID != "" {
		teamUUID, _ = types.StringToUUID(*req.AssignedTeamID)
	}

	var agentUUID pgtype.UUID
	if req.AssignedAgentID != nil && *req.AssignedAgentID != "" {
		agentUUID, _ = types.StringToUUID(*req.AssignedAgentID)
	}

	var createdByUUID pgtype.UUID
	if claims != nil {
		createdByUUID, _ = types.StringToUUID(claims.UserID)
	}

	// Default Open Status
	statusID := pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x01}, Valid: true}

	ticket, err := h.queries.CreateTicket(r.Context(), db.CreateTicketParams{
		ID:                  ticketID,
		Subject:             req.Subject,
		DescriptionHtml:     "<p>" + req.Description + "</p>",
		DescriptionText:     req.Description,
		StatusID:            statusID,
		StatusCategory:      "open",
		PriorityID:          priorityUUID,
		PriorityWeight:      2,
		TypeID:              typeUUID,
		ContactID:           contact.ID,
		OrganizationID:      contact.OrganizationID,
		AssignedAgentID:     agentUUID,
		AssignedTeamID:      teamUUID,
		Source:              "agent",
		CustomData:          []byte("{}"),
		CreatedBy:           createdByUUID,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to create ticket"}`, http.StatusInternalServerError)
		return
	}

	// Transactional outbox event
	_ = outbox.Publish(r.Context(), h.queries, "ticket", ticket.ID, outbox.EventTicketCreated, map[string]any{
		"ticket_id":    types.UUIDToString(ticket.ID),
		"reference_no": ticket.ReferenceNo,
		"subject":      ticket.Subject,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ticket)
}

type UpdateTicketFieldsRequest struct {
	StatusID        *string `json:"status_id,omitempty"`
	StatusKey       *string `json:"status_key,omitempty"`
	StatusCategory  *string `json:"status_category,omitempty"`
	PriorityID      *string `json:"priority_id,omitempty"`
	PriorityKey     *string `json:"priority_key,omitempty"`
	TypeID          *string `json:"type_id,omitempty"`
	AssignedAgentID *string `json:"assigned_agent_id,omitempty"`
	AssignedTeamID  *string `json:"assigned_team_id,omitempty"`
}

// UpdateTicket handles PATCH /api/v1/app/tickets/{id}
func (h *AppTicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := types.StringToUUID(ticketIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid ticket uuid"}`, http.StatusBadRequest)
		return
	}

	var req UpdateTicketFieldsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	currentTicket, err := h.queries.GetTicketDetail(r.Context(), ticketID)
	if err != nil {
		http.Error(w, `{"error":"not_found","message":"ticket not found"}`, http.StatusNotFound)
		return
	}

	var statusUUID pgtype.UUID = currentTicket.StatusID
	var statusCategory pgtype.Text = pgtype.Text{String: currentTicket.StatusCategory, Valid: true}
	var priorityUUID pgtype.UUID = currentTicket.PriorityID
	var typeUUID pgtype.UUID = currentTicket.TypeID
	var agentUUID pgtype.UUID = currentTicket.AssignedAgentID
	var teamUUID pgtype.UUID = currentTicket.AssignedTeamID

	now := time.Now().UTC()
	nowTz := types.TimeToTimestamptz(now)
	var resolvedAt, closedAt pgtype.Timestamptz

	if req.StatusKey != nil || req.StatusCategory != nil || req.StatusID != nil {
		statusVal := ""
		if req.StatusKey != nil && *req.StatusKey != "" {
			statusVal = strings.ToLower(strings.TrimSpace(*req.StatusKey))
		} else if req.StatusCategory != nil && *req.StatusCategory != "" {
			statusVal = strings.ToLower(strings.TrimSpace(*req.StatusCategory))
		} else if req.StatusID != nil && *req.StatusID != "" {
			statusVal = strings.ToLower(strings.TrimSpace(*req.StatusID))
		}

		switch statusVal {
		case "new", "018e0000-0000-7000-8000-000000000101":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x01}, Valid: true}
			statusCategory = pgtype.Text{String: "open", Valid: true}
		case "open", "018e0000-0000-7000-8000-000000000102":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x02}, Valid: true}
			statusCategory = pgtype.Text{String: "open", Valid: true}
		case "pending", "018e0000-0000-7000-8000-000000000103":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x03}, Valid: true}
			statusCategory = pgtype.Text{String: "pending", Valid: true}
		case "on_hold", "paused", "018e0000-0000-7000-8000-000000000104":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x04}, Valid: true}
			statusCategory = pgtype.Text{String: "paused", Valid: true}
		case "resolved", "018e0000-0000-7000-8000-000000000105":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x05}, Valid: true}
			statusCategory = pgtype.Text{String: "resolved", Valid: true}
			resolvedAt = nowTz
		case "closed", "018e0000-0000-7000-8000-000000000106":
			statusUUID = pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x06}, Valid: true}
			statusCategory = pgtype.Text{String: "closed", Valid: true}
			closedAt = nowTz
		default:
			if u, err := types.StringToUUID(statusVal); err == nil {
				statusUUID = u
			}
		}
	}

	if req.PriorityID != nil {
		if u, err := types.StringToUUID(*req.PriorityID); err == nil {
			priorityUUID = u
		}
	}
	if req.TypeID != nil {
		if *req.TypeID == "" {
			typeUUID = pgtype.UUID{Valid: false}
		} else if u, err := types.StringToUUID(*req.TypeID); err == nil {
			typeUUID = u
		}
	}
	if req.AssignedAgentID != nil {
		if *req.AssignedAgentID == "" {
			agentUUID = pgtype.UUID{Valid: false}
		} else if u, err := types.StringToUUID(*req.AssignedAgentID); err == nil {
			agentUUID = u
		}
	}
	if req.AssignedTeamID != nil {
		if *req.AssignedTeamID == "" {
			teamUUID = pgtype.UUID{Valid: false}
		} else if u, err := types.StringToUUID(*req.AssignedTeamID); err == nil {
			teamUUID = u
		}
	}

	updated, err := h.queries.UpdateTicketFields(r.Context(), db.UpdateTicketFieldsParams{
		ID:              ticketID,
		StatusID:        statusUUID,
		StatusCategory:  statusCategory,
		PriorityID:      priorityUUID,
		TypeID:          typeUUID,
		AssignedAgentID: agentUUID,
		AssignedTeamID:  teamUUID,
		ResolvedAt:      resolvedAt,
		ClosedAt:        closedAt,
	})
	if err != nil {
		slog.Error("UpdateTicketFields failed", "error", err, "ticket_id", ticketIDStr)
		http.Error(w, `{"error":"internal_error","message":"failed to update ticket"}`, http.StatusInternalServerError)
		return
	}

	// Publish update event
	_ = outbox.Publish(r.Context(), h.queries, "ticket", ticketID, outbox.EventTicketUpdated, map[string]any{
		"ticket_id": types.UUIDToString(ticketID),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}
