package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

// EventType constants
const (
	EventTicketCreated       = "ticket.created"
	EventTicketUpdated       = "ticket.updated"
	EventReplyReceived       = "reply.received"
	EventInternalNoteAdded   = "note.added"
	EventStatusChanged       = "status.changed"
	EventAssignmentChanged   = "assignment.changed"
	EventSLABreached         = "sla.breached"
	EventCustomerFeedback    = "feedback.received"
	EventOutboundEmailQueued = "email.outbound_queued"
)

type HandlerFunc func(ctx context.Context, item db.Outbox) error

type Worker struct {
	queries  db.Querier
	handlers map[string][]HandlerFunc
	interval time.Duration
	stopCh   chan struct{}
}

func NewWorker(queries db.Querier, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	return &Worker{
		queries:  queries,
		handlers: make(map[string][]HandlerFunc),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (w *Worker) RegisterHandler(event string, handler HandlerFunc) {
	w.handlers[event] = append(w.handlers[event], handler)
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.ProcessBatch(ctx)
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) ProcessBatch(ctx context.Context) {
	items, err := w.queries.ClaimUnpublishedOutboxBatch(ctx, 25)
	if err != nil {
		slog.Error("Failed to claim outbox batch", "error", err)
		return
	}

	for _, item := range items {
		w.processItem(ctx, item)
	}
}

func (w *Worker) processItem(ctx context.Context, item db.Outbox) {
	handlers := w.handlers[item.Event]
	var lastErr error

	for _, h := range handlers {
		if err := h(ctx, item); err != nil {
			lastErr = err
			slog.Error("Outbox handler failed", "event", item.Event, "id", types.UUIDToString(item.ID), "error", err)
			break
		}
	}

	if lastErr != nil {
		_ = w.queries.MarkOutboxFailed(ctx, db.MarkOutboxFailedParams{
			ID:        item.ID,
			LastError: pgtype.Text{String: lastErr.Error(), Valid: true},
		})
	} else {
		_ = w.queries.MarkOutboxPublished(ctx, item.ID)
	}
}

// Publish creates an outbox record inside the active transaction/query executor
func Publish(
	ctx context.Context,
	q db.Querier,
	aggregate string,
	aggregateID pgtype.UUID,
	event string,
	payload any,
) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	_, err = q.InsertOutboxMessage(ctx, db.InsertOutboxMessageParams{
		ID:          types.NewUUIDv7(),
		Aggregate:   aggregate,
		AggregateID: aggregateID,
		Event:       event,
		Payload:     payloadBytes,
	})
	return err
}
