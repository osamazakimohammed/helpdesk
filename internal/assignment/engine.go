package assignment

import (
	"context"
	"fmt"
	"time"

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

type AssignmentResult struct {
	AssignedTeamID  pgtype.UUID
	AssignedAgentID pgtype.UUID
	RuleMatched     bool
	RuleName        string
}

func (e *Engine) EvaluateAssignment(
	ctx context.Context,
	trigger string,
	ticketTypeID pgtype.UUID,
	priorityWeight int32,
	requiredSkills []string,
) (*AssignmentResult, error) {
	rules, err := e.queries.ListActiveAssignmentRules(ctx, trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignment rules: %w", err)
	}

	if len(rules) == 0 {
		return &AssignmentResult{RuleMatched: false}, nil
	}

	agents, err := e.queries.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	agentMap := make(map[string]db.ListAgentsRow)
	for _, a := range agents {
		agentMap[types.UUIDToString(a.ID)] = a
	}

	now := time.Now().UTC()
	currentWeekday := int(now.Weekday())

	for _, rule := range rules {
		// 1. Check weekday filter
		weekdayMatch := false
		for _, w := range rule.ActiveWeekdays {
			if int(w) == currentWeekday {
				weekdayMatch = true
				break
			}
		}
		if !weekdayMatch && len(rule.ActiveWeekdays) > 0 {
			continue
		}

		// 2. Filter eligible candidate agents
		var eligibleAgents []db.ListAgentsRow
		if len(rule.CandidateAgentIds) > 0 {
			for _, candidateID := range rule.CandidateAgentIds {
				candStr := types.UUIDToString(candidateID)
				if a, ok := agentMap[candStr]; ok {
					if rule.RespectAvailability && !a.IsAvailable {
						continue
					}
					if rule.RespectCapacity && a.OpenTicketCount >= int64(a.MaxConcurrentTickets) {
						continue
					}
					eligibleAgents = append(eligibleAgents, a)
				}
			}
		} else {
			for _, a := range agents {
				if rule.RespectAvailability && !a.IsAvailable {
					continue
				}
				if rule.RespectCapacity && a.OpenTicketCount >= int64(a.MaxConcurrentTickets) {
					continue
				}
				eligibleAgents = append(eligibleAgents, a)
			}
		}

		if len(eligibleAgents) == 0 {
			// If team is specified, assign to team even if no individual agent is available
			if rule.TargetTeamID.Valid {
				return &AssignmentResult{
					AssignedTeamID: rule.TargetTeamID,
					RuleMatched:    true,
					RuleName:       rule.Name,
				}, nil
			}
			continue
		}

		// 3. Execute strategy
		var selectedAgent db.ListAgentsRow

		switch rule.Strategy {
		case "least_open":
			minCount := eligibleAgents[0].OpenTicketCount
			selectedAgent = eligibleAgents[0]
			for _, a := range eligibleAgents[1:] {
				if a.OpenTicketCount < minCount {
					minCount = a.OpenTicketCount
					selectedAgent = a
				}
			}

		case "round_robin":
			lastIDStr := types.UUIDToString(rule.LastAssignedAgentID)
			foundIndex := -1
			for i, a := range eligibleAgents {
				if types.UUIDToString(a.ID) == lastIDStr {
					foundIndex = i
					break
				}
			}
			nextIndex := (foundIndex + 1) % len(eligibleAgents)
			selectedAgent = eligibleAgents[nextIndex]

		case "skill_match":
			maxMatch := -1
			selectedAgent = eligibleAgents[0]
			for _, a := range eligibleAgents {
				matchCount := 0
				for _, req := range requiredSkills {
					for _, sk := range a.Skills {
						if sk == req {
							matchCount++
						}
					}
				}
				if matchCount > maxMatch {
					maxMatch = matchCount
					selectedAgent = a
				}
			}

		default:
			selectedAgent = eligibleAgents[0]
		}

		// Update last assigned agent on rule
		_ = e.queries.UpdateAssignmentRuleLastAgent(ctx, db.UpdateAssignmentRuleLastAgentParams{
			ID:                  rule.ID,
			LastAssignedAgentID: selectedAgent.ID,
		})

		return &AssignmentResult{
			AssignedTeamID:  rule.TargetTeamID,
			AssignedAgentID: selectedAgent.ID,
			RuleMatched:     true,
			RuleName:        rule.Name,
		}, nil
	}

	return &AssignmentResult{RuleMatched: false}, nil
}
