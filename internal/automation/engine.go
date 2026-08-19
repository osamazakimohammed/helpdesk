package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

type Engine struct {
	queries db.Querier
}

func NewEngine(queries db.Querier) *Engine {
	return &Engine{queries: queries}
}

type ActionConfig struct {
	SetStatusID   *string `json:"set_status_id,omitempty"`
	SetPriorityID *string `json:"set_priority_id,omitempty"`
	SetTeamID     *string `json:"set_team_id,omitempty"`
	SetAgentID    *string `json:"set_agent_id,omitempty"`
	AddTagID      *string `json:"add_tag_id,omitempty"`
	AddNote       *string `json:"add_internal_note,omitempty"`
}

func (e *Engine) ProcessTrigger(
	ctx context.Context,
	trigger string,
	ticket db.GetTicketDetailRow,
) error {
	rules, err := e.queries.ListActiveAutomationRulesByTrigger(ctx, trigger)
	if err != nil {
		return fmt.Errorf("failed to list automation rules for trigger %s: %w", trigger, err)
	}

	for _, rule := range rules {
		// Evaluate conditions (JSONB match)
		if !e.evaluateConditions(rule.MatchConditions, ticket) {
			continue
		}

		// Execute actions
		actionsApplied, err := e.executeActions(ctx, rule.Actions, ticket)
		status := "success"
		var errStr pgtype.Text
		if err != nil {
			status = "failed"
			errStr = pgtype.Text{String: err.Error(), Valid: true}
			slog.Error("Automation rule execution failed", "rule_id", types.UUIDToString(rule.ID), "ticket_id", types.UUIDToString(ticket.ID), "error", err)
		}

		actionsBytes, _ := json.Marshal(actionsApplied)
		_ = e.queries.RecordAutomationRun(ctx, db.RecordAutomationRunParams{
			ID:             types.NewUUIDv7(),
			RuleID:         rule.ID,
			TicketID:       ticket.ID,
			Status:         status,
			Error:          errStr,
			ActionsApplied: actionsBytes,
		})

		if rule.RunOncePerTicket {
			break
		}
	}

	return nil
}

func (e *Engine) evaluateConditions(matchCond []byte, ticket db.GetTicketDetailRow) bool {
	if len(matchCond) == 0 || string(matchCond) == "{}" {
		return true
	}

	var conds map[string]any
	if err := json.Unmarshal(matchCond, &conds); err != nil {
		return true
	}

	if statusKey, ok := conds["status_key"].(string); ok {
		if ticket.StatusKey != statusKey {
			return false
		}
	}

	if priorityKey, ok := conds["priority_key"].(string); ok {
		if ticket.PriorityKey != priorityKey {
			return false
		}
	}

	return true
}

func (e *Engine) executeActions(
	ctx context.Context,
	actionsJson []byte,
	ticket db.GetTicketDetailRow,
) (map[string]any, error) {
	var actions ActionConfig
	if err := json.Unmarshal(actionsJson, &actions); err != nil {
		return nil, fmt.Errorf("invalid action config: %w", err)
	}

	applied := make(map[string]any)

	// Update Ticket fields if specified
	var newStatusID, newPriorityID, newTeamID, newAgentID pgtype.UUID
	needsUpdate := false

	if actions.SetStatusID != nil {
		if u, err := types.StringToUUID(*actions.SetStatusID); err == nil {
			newStatusID = u
			needsUpdate = true
			applied["status_id"] = *actions.SetStatusID
		}
	}

	if actions.SetPriorityID != nil {
		if u, err := types.StringToUUID(*actions.SetPriorityID); err == nil {
			newPriorityID = u
			needsUpdate = true
			applied["priority_id"] = *actions.SetPriorityID
		}
	}

	if actions.SetTeamID != nil {
		if u, err := types.StringToUUID(*actions.SetTeamID); err == nil {
			newTeamID = u
			needsUpdate = true
			applied["team_id"] = *actions.SetTeamID
		}
	}

	if actions.SetAgentID != nil {
		if u, err := types.StringToUUID(*actions.SetAgentID); err == nil {
			newAgentID = u
			needsUpdate = true
			applied["agent_id"] = *actions.SetAgentID
		}
	}

	if needsUpdate {
		_, err := e.queries.UpdateTicketFields(ctx, db.UpdateTicketFieldsParams{
			ID:              ticket.ID,
			StatusID:        newStatusID,
			PriorityID:      newPriorityID,
			AssignedTeamID:  newTeamID,
			AssignedAgentID: newAgentID,
		})
		if err != nil {
			return applied, fmt.Errorf("failed to update ticket fields in automation: %w", err)
		}
	}

	if actions.AddTagID != nil {
		if tagUUID, err := types.StringToUUID(*actions.AddTagID); err == nil {
			_ = e.queries.AddTicketTag(ctx, db.AddTicketTagParams{
				TicketID: ticket.ID,
				TagID:    tagUUID,
			})
			applied["tag_added"] = *actions.AddTagID
		}
	}

	if actions.AddNote != nil && *actions.AddNote != "" {
		_, _ = e.queries.CreateTicketEvent(ctx, db.CreateTicketEventParams{
			ID:          types.NewUUIDv7(),
			TicketID:    ticket.ID,
			Kind:        "automation",
			Visibility:  "internal",
			AuthorType:  "automation",
			BodyHtml:    fmt.Sprintf("<p>%s</p>", *actions.AddNote),
			BodyText:    *actions.AddNote,
			Metadata:    []byte("{}"),
			OccurredAt:  types.TimeToTimestamptz(types.TimestamptzToNullTime(ticket.CreatedAt).UTC()),
		})
		applied["note_added"] = true
	}

	return applied, nil
}
