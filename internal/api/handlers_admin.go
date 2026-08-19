package api

import (
	"encoding/json"
	"net/http"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

type AdminHandler struct {
	queries db.Querier
}

func NewAdminHandler(queries db.Querier) *AdminHandler {
	return &AdminHandler{queries: queries}
}

// ListSLAPolicies handles GET /api/v1/admin/sla-policies
func (h *AdminHandler) ListSLAPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.queries.ListActiveSLAPolicies(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch sla policies"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(policies)
}

// ListFieldDefinitions handles GET /api/v1/admin/fields
func (h *AdminHandler) ListFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	fields, err := h.queries.ListAllFieldDefinitions(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch field definitions"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fields)
}

// ListMailAccounts handles GET /api/v1/admin/mail-accounts
func (h *AdminHandler) ListMailAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.queries.ListMailAccounts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch mail accounts"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(accounts)
}

// ListAutomationRules handles GET /api/v1/admin/automation-rules
func (h *AdminHandler) ListAutomationRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.queries.ListAutomationRules(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch automation rules"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rules)
}

// ListAssignmentRules handles GET /api/v1/admin/assignment-rules
func (h *AdminHandler) ListAssignmentRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.queries.ListAssignmentRules(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch assignment rules"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rules)
}

// ListAuditLogs handles GET /api/v1/admin/audit-logs
func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	entity := q.Get("entity")
	var entityID pgtype.UUID
	if idStr := q.Get("entity_id"); idStr != "" {
		if u, err := types.StringToUUID(idStr); err == nil {
			entityID = u
		}
	}

	logs, err := h.queries.ListAuditLogsKeyset(r.Context(), db.ListAuditLogsKeysetParams{
		Column1: entityID,
		Column2: entity,
		Limit:   50,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch audit logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": logs})
}

// ListAgents handles GET /api/v1/admin/agents
func (h *AdminHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.queries.ListAgents(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to list agents"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

// ListTeams handles GET /api/v1/admin/teams
func (h *AdminHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.queries.ListTeams(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to list teams"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(teams)
}
