package sla

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

// BusinessHourSlot represents daily active hours
type BusinessHourSlot struct {
	Weekday   int    `json:"weekday"`    // 0 = Sunday, 1 = Monday, ... 6 = Saturday
	StartTime string `json:"start_time"` // "09:00:00"
	EndTime   string `json:"end_time"`   // "17:00:00"
}

// HolidayItem represents a calendar holiday
type HolidayItem struct {
	Date              string `json:"date"` // "2026-12-25"
	Label             string `json:"label"`
	IsRecurringAnnual bool   `json:"is_recurring_annual"`
}

// YearScheduleTable contains the precomputed minute-by-minute business calendar
type YearScheduleTable struct {
	Year                      int
	Location                  *time.Location
	StartUTC                  time.Time
	EndUTC                    time.Time
	minuteIsBusiness          []bool
	cumulativeBusinessMinutes []int
	businessMinuteToUTC       []time.Time
}

// Engine manages SLA evaluations and precomputed business schedules
type Engine struct {
	mu         sync.RWMutex
	queries    db.Querier
	yearTables map[string]*YearScheduleTable // key: calendarID_year
}

func NewEngine(queries db.Querier) *Engine {
	return &Engine{
		queries:    queries,
		yearTables: make(map[string]*YearScheduleTable),
	}
}

// PrecomputeYear builds the O(1) fast lookup table for a given business hours & holiday calendar
func (e *Engine) PrecomputeYear(
	ctx context.Context,
	key string,
	year int,
	tzName string,
	slots []BusinessHourSlot,
	holidays []HolidayItem,
) (*YearScheduleTable, error) {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, loc)

	totalMinutes := int(endOfYear.Sub(startOfYear).Minutes())
	minuteIsBusiness := make([]bool, totalMinutes)
	cumulativeBusinessMinutes := make([]int, totalMinutes)
	var businessMinuteToUTC []time.Time

	// Parse holiday map for quick lookup
	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Date] = true
	}

	// Slot lookup table: weekday -> intervals in minutes from midnight
	type interval struct{ start, end int }
	slotsByWeekday := make(map[int][]interval)
	for _, s := range slots {
		var sh, sm, ss, eh, em, es int
		_, _ = fmt.Sscanf(s.StartTime, "%d:%d:%d", &sh, &sm, &ss)
		_, _ = fmt.Sscanf(s.EndTime, "%d:%d:%d", &eh, &em, &es)
		startMin := sh*60 + sm
		endMin := eh*60 + em
		if endMin > startMin {
			slotsByWeekday[s.Weekday] = append(slotsByWeekday[s.Weekday], interval{start: startMin, end: endMin})
		}
	}

	count := 0
	for m := 0; m < totalMinutes; m++ {
		tLocal := startOfYear.Add(time.Duration(m) * time.Minute)
		weekday := int(tLocal.Weekday())
		dateStr := tLocal.Format("2006-01-02")
		minOfDay := tLocal.Hour()*60 + tLocal.Minute()

		isWorking := false
		if !holidayMap[dateStr] {
			if intervals, ok := slotsByWeekday[weekday]; ok {
				for _, iv := range intervals {
					if minOfDay >= iv.start && minOfDay < iv.end {
						isWorking = true
						break
					}
				}
			}
		}

		minuteIsBusiness[m] = isWorking
		if isWorking {
			count++
			businessMinuteToUTC = append(businessMinuteToUTC, tLocal.UTC())
		}
		cumulativeBusinessMinutes[m] = count
	}

	table := &YearScheduleTable{
		Year:                      year,
		Location:                  loc,
		StartUTC:                  startOfYear.UTC(),
		EndUTC:                    endOfYear.UTC(),
		minuteIsBusiness:          minuteIsBusiness,
		cumulativeBusinessMinutes: cumulativeBusinessMinutes,
		businessMinuteToUTC:       businessMinuteToUTC,
	}

	e.mu.Lock()
	e.yearTables[fmt.Sprintf("%s_%d", key, year)] = table
	e.mu.Unlock()

	return table, nil
}

// ComputeDueTimestamp calculates the exact target UTC time using O(1) table lookup
func (e *Engine) ComputeDueTimestamp(
	ctx context.Context,
	table *YearScheduleTable,
	startUTC time.Time,
	targetDurationMinutes int,
) time.Time {
	if table == nil || len(table.businessMinuteToUTC) == 0 {
		return startUTC.Add(time.Duration(targetDurationMinutes) * time.Minute)
	}

	startLocal := startUTC.In(table.Location)
	if startLocal.Year() != table.Year {
		return startUTC.Add(time.Duration(targetDurationMinutes) * time.Minute)
	}

	startOfYear := time.Date(table.Year, 1, 1, 0, 0, 0, 0, table.Location)
	m := int(startLocal.Sub(startOfYear).Minutes())
	if m < 0 {
		m = 0
	}
	if m >= len(table.cumulativeBusinessMinutes) {
		m = len(table.cumulativeBusinessMinutes) - 1
	}

	currentBusinessMinute := table.cumulativeBusinessMinutes[m]
	targetBusinessMinute := currentBusinessMinute + targetDurationMinutes

	if targetBusinessMinute <= 0 {
		return startUTC
	}

	// Direct O(1) array lookup
	targetIndex := targetBusinessMinute - 1
	if targetIndex < len(table.businessMinuteToUTC) {
		return table.businessMinuteToUTC[targetIndex]
	}

	lastTime := table.businessMinuteToUTC[len(table.businessMinuteToUTC)-1]
	overflow := targetIndex - (len(table.businessMinuteToUTC) - 1)
	return lastTime.Add(time.Duration(overflow) * time.Minute)
}

// EvaluateTicketSLA determines SLA target due dates for a given ticket
type SLAResult struct {
	PolicyID           pgtype.UUID
	PolicyName         string
	FirstResponseDueAt *time.Time
	ResolutionDueAt    *time.Time
}

func (e *Engine) EvaluateTicketSLA(
	ctx context.Context,
	ticketCreatedAt time.Time,
	priorityID pgtype.UUID,
) (*SLAResult, error) {
	if e.queries == nil {
		return nil, nil
	}
	policies, err := e.queries.ListActiveSLAPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active SLA policies: %w", err)
	}

	if len(policies) == 0 {
		return nil, nil
	}

	policy := policies[0]

	var targets []struct {
		PriorityID           string `json:"priority_id"`
		FirstResponseMinutes int    `json:"first_response_minutes"`
		ResolutionMinutes    int    `json:"resolution_minutes"`
		NextResponseMinutes  *int   `json:"next_response_minutes"`
	}

	var rawBytes []byte
	if b, ok := policy.TargetsJson.([]byte); ok {
		rawBytes = b
	} else {
		rawBytes, _ = json.Marshal(policy.TargetsJson)
	}

	if err := json.Unmarshal(rawBytes, &targets); err != nil {
		return nil, fmt.Errorf("failed to parse SLA targets: %w", err)
	}

	priorityStr := types.UUIDToString(priorityID)
	var firstRespMin, resMin int

	for _, t := range targets {
		if t.PriorityID == priorityStr {
			firstRespMin = t.FirstResponseMinutes
			resMin = t.ResolutionMinutes
			break
		}
	}

	if firstRespMin == 0 && resMin == 0 {
		firstRespMin = 240
		resMin = 1440
	}

	res := &SLAResult{
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
	}

	var table *YearScheduleTable
	if policy.BusinessHoursID.Valid {
		e.mu.RLock()
		table = e.yearTables[fmt.Sprintf("%s_%d", types.UUIDToString(policy.BusinessHoursID), ticketCreatedAt.Year())]
		e.mu.RUnlock()
	}

	if firstRespMin > 0 {
		due := e.ComputeDueTimestamp(ctx, table, ticketCreatedAt, firstRespMin)
		res.FirstResponseDueAt = &due
	}

	if resMin > 0 {
		due := e.ComputeDueTimestamp(ctx, table, ticketCreatedAt, resMin)
		res.ResolutionDueAt = &due
	}

	return res, nil
}

// CalculatePausedResumeShift computes shifted due dates when ticket resumes from pause
func CalculatePausedResumeShift(
	pausedAt time.Time,
	resumedAt time.Time,
	firstResponseDue *time.Time,
	resolutionDue *time.Time,
	prevPausedMs int64,
) (newFirstResponseDue *time.Time, newResolutionDue *time.Time, newPausedMs int64) {
	elapsed := resumedAt.Sub(pausedAt)
	if elapsed < 0 {
		elapsed = 0
	}

	newPausedMs = prevPausedMs + elapsed.Milliseconds()

	if firstResponseDue != nil && !firstResponseDue.IsZero() {
		shifted := firstResponseDue.Add(elapsed)
		newFirstResponseDue = &shifted
	}

	if resolutionDue != nil && !resolutionDue.IsZero() {
		shifted := resolutionDue.Add(elapsed)
		newResolutionDue = &shifted
	}

	return
}
