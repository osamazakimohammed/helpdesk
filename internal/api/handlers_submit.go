package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/outbox"
	"helpdesk/internal/storage"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

type SubmitHandler struct {
	queries db.Querier
	storage storage.Storage
}

func NewSubmitHandler(queries db.Querier, storage storage.Storage) *SubmitHandler {
	return &SubmitHandler{
		queries: queries,
		storage: storage,
	}
}

// GetIntakeFields handles GET /api/v1/submit/fields
func (h *SubmitHandler) GetIntakeFields(w http.ResponseWriter, r *http.Request) {
	fields, err := h.queries.ListFieldDefinitions(r.Context(), "ticket")
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch custom fields"}`, http.StatusInternalServerError)
		return
	}

	var portalFields []db.FieldDefinitions
	for _, f := range fields {
		if f.ShowInPortalForm && !f.IsAgentOnly {
			portalFields = append(portalFields, f)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(portalFields)
}

type AnonymousTicketRequest struct {
	FullName     string         `json:"full_name"`
	Email        string         `json:"email"`
	Subject      string         `json:"subject"`
	Description  string         `json:"description"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
	CaptchaToken string         `json:"captcha_token,omitempty"`
}

// SubmitTicket handles POST /api/v1/submit/ticket
func (h *SubmitHandler) SubmitTicket(w http.ResponseWriter, r *http.Request) {
	var req AnonymousTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid json payload"}`, http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Description = strings.TrimSpace(req.Description)

	if req.Email == "" || req.Subject == "" || req.Description == "" {
		http.Error(w, `{"error":"bad_request","message":"email, subject, and description are required"}`, http.StatusBadRequest)
		return
	}

	// 1. Resolve Contact & Organization
	contact, err := h.queries.GetContactByEmail(r.Context(), req.Email)
	if err != nil || !contact.ID.Valid {
		name := req.FullName
		if name == "" {
			name = req.Email
		}

		parts := strings.Split(req.Email, "@")
		var orgID pgtype.UUID
		if len(parts) == 2 {
			domain := strings.ToLower(parts[1])
			slug := strings.ReplaceAll(domain, ".", "-")
			org, err := h.queries.GetOrCreateOrganizationByDomain(r.Context(), db.GetOrCreateOrganizationByDomainParams{
				ID:     types.NewUUIDv7(),
				Name:   domain,
				Slug:   slug,
				Domain: pgtype.Text{String: domain, Valid: true},
			})
			if err != nil {
				slog.Warn("Organization creation error", "error", err)
			} else {
				orgID = org.ID
			}
		}

		contact, err = h.queries.CreateContact(r.Context(), db.CreateContactParams{
			ID:             types.NewUUIDv7(),
			OrganizationID: orgID,
			PrimaryEmail:   req.Email,
			FullName:       name,
			Locale:         "en",
			Timezone:       "UTC",
			IsVerified:     false,
			CustomData:     []byte("{}"),
		})
		if err != nil {
			slog.Error("CreateContact failed", "error", err, "email", req.Email)
			http.Error(w, `{"error":"internal_error","message":"failed to register contact: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	ticketID := types.NewUUIDv7()
	now := time.Now().UTC()
	priorityUUID := pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x02, 0x02}, Valid: true} // Medium
	statusID := pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x01}, Valid: true}     // New

	customDataBytes := []byte("{}")
	if len(req.CustomFields) > 0 {
		customDataBytes, _ = json.Marshal(req.CustomFields)
	}

	ticket, err := h.queries.CreateTicket(r.Context(), db.CreateTicketParams{
		ID:              ticketID,
		Subject:         req.Subject,
		DescriptionHtml: "<p>" + req.Description + "</p>",
		DescriptionText: req.Description,
		StatusID:        statusID,
		StatusCategory:  "open",
		PriorityID:      priorityUUID,
		PriorityWeight:  2,
		ContactID:       contact.ID,
		OrganizationID:  contact.OrganizationID,
		Source:          "form",
		CustomData:      customDataBytes,
	})
	if err != nil {
		slog.Error("CreateTicket failed", "error", err)
		http.Error(w, `{"error":"internal_error","message":"failed to create ticket"}`, http.StatusInternalServerError)
		return
	}

	// Create initial event
	_, _ = h.queries.CreateTicketEvent(r.Context(), db.CreateTicketEventParams{
		ID:             types.NewUUIDv7(),
		TicketID:       ticket.ID,
		Kind:           "portal_reply",
		Visibility:     "public",
		AuthorType:     "contact",
		AuthorID:       contact.ID,
		BodyHtml:       "<p>" + req.Description + "</p>",
		BodyText:       req.Description,
		Metadata:       []byte("{}"),
		DeliveryStatus: "delivered",
		OccurredAt:     types.TimeToTimestamptz(now),
	})

	_ = outbox.Publish(r.Context(), h.queries, "ticket", ticket.ID, outbox.EventTicketCreated, map[string]any{
		"ticket_id":    types.UUIDToString(ticket.ID),
		"reference_no": ticket.ReferenceNo,
		"subject":      ticket.Subject,
		"from_email":   req.Email,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":      true,
		"ticket_id":    types.UUIDToString(ticket.ID),
		"reference_no": ticket.ReferenceNo,
		"message":      "Ticket submitted successfully. You can track it using your reference number.",
	})
}
