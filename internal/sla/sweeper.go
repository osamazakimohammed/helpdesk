package sla

import (
	"context"
	"log/slog"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
)

type Sweeper struct {
	queries  db.Querier
	interval time.Duration
	stopCh   chan struct{}
}

func NewSweeper(queries db.Querier, interval time.Duration) *Sweeper {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Sweeper{
		queries:  queries,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (s *Sweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.RunSweep(ctx)
			}
		}
	}()
}

func (s *Sweeper) Stop() {
	close(s.stopCh)
}

func (s *Sweeper) RunSweep(ctx context.Context) {
	now := time.Now().UTC()
	nowTz := types.TimeToTimestamptz(now)

	// 1. Scan for first response breaches using the partial index
	firstRespBreaches, err := s.queries.ScanBreachingFirstResponseTickets(ctx, db.ScanBreachingFirstResponseTicketsParams{
		FirstResponseDueAt: nowTz,
		Limit:              100,
	})
	if err != nil {
		slog.Error("SLA Sweeper failed to scan first response breaches", "error", err)
	} else {
		for _, t := range firstRespBreaches {
			slog.Warn("SLA First Response Breached", "ticket_id", types.UUIDToString(t.ID), "reference_no", t.ReferenceNo)
			_ = s.queries.MarkFirstResponseBreached(ctx, t.ID)
		}
	}

	// 2. Scan for resolution breaches using the partial index
	resBreaches, err := s.queries.ScanBreachingResolutionTickets(ctx, db.ScanBreachingResolutionTicketsParams{
		ResolutionDueAt: nowTz,
		Limit:           100,
	})
	if err != nil {
		slog.Error("SLA Sweeper failed to scan resolution breaches", "error", err)
	} else {
		for _, t := range resBreaches {
			slog.Warn("SLA Resolution Breached", "ticket_id", types.UUIDToString(t.ID), "reference_no", t.ReferenceNo)
			_ = s.queries.MarkResolutionBreached(ctx, t.ID)
		}
	}
}
