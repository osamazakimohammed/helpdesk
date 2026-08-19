package sla

import (
	"context"
	"testing"
	"time"
)

func TestPrecomputedBusinessMinutes(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	// 9 to 17 UTC Monday through Friday
	slots := []BusinessHourSlot{
		{Weekday: 1, StartTime: "09:00:00", EndTime: "17:00:00"},
		{Weekday: 2, StartTime: "09:00:00", EndTime: "17:00:00"},
		{Weekday: 3, StartTime: "09:00:00", EndTime: "17:00:00"},
		{Weekday: 4, StartTime: "09:00:00", EndTime: "17:00:00"},
		{Weekday: 5, StartTime: "09:00:00", EndTime: "17:00:00"},
	}

	holidays := []HolidayItem{
		{Date: "2026-01-01", Label: "New Year's Day", IsRecurringAnnual: true},
	}

	table, err := engine.PrecomputeYear(ctx, "test_cal", 2026, "UTC", slots, holidays)
	if err != nil {
		t.Fatalf("failed to precompute year: %v", err)
	}

	if len(table.businessMinuteToUTC) == 0 {
		t.Fatal("expected precomputed business minutes, got 0")
	}

	// Friday 2026-01-02 at 16:00 UTC (1 hour before closing)
	fridayStart := time.Date(2026, 1, 2, 16, 0, 0, 0, time.UTC)
	// Target: 120 business minutes (2 hours). Should skip weekend and land on Monday 2026-01-05 at 10:00 UTC!
	dueAt := engine.ComputeDueTimestamp(ctx, table, fridayStart, 120)

	expectedDue := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	if !dueAt.Equal(expectedDue) {
		t.Errorf("expected due date %v, got %v", expectedDue, dueAt)
	}
}

func TestSLAForwardShiftOnResume(t *testing.T) {
	pausedAt := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	resumedAt := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC) // Paused for 2 hours

	origFirstResp := time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC)
	origResolution := time.Date(2026, 1, 6, 17, 0, 0, 0, time.UTC)

	newFirstResp, newResolution, pausedMs := CalculatePausedResumeShift(
		pausedAt, resumedAt, &origFirstResp, &origResolution, 0,
	)

	expectedMs := int64(2 * 3600 * 1000)
	if pausedMs != expectedMs {
		t.Errorf("expected pausedMs %d, got %d", expectedMs, pausedMs)
	}

	expectedFirstResp := time.Date(2026, 1, 5, 16, 0, 0, 0, time.UTC)
	if !newFirstResp.Equal(expectedFirstResp) {
		t.Errorf("expected shifted first response %v, got %v", expectedFirstResp, *newFirstResp)
	}

	expectedResolution := time.Date(2026, 1, 6, 19, 0, 0, 0, time.UTC)
	if !newResolution.Equal(expectedResolution) {
		t.Errorf("expected shifted resolution %v, got %v", expectedResolution, *newResolution)
	}
}
