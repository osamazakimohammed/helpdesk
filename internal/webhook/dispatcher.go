package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
)

type Dispatcher struct {
	queries    db.Querier
	httpClient *http.Client
}

func NewDispatcher(queries db.Querier) *Dispatcher {
	return &Dispatcher{
		queries: queries,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ComputeSignature calculates the HMAC-SHA256 hex string of the payload
func ComputeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// DispatchEvent finds matching webhooks and delivers the signed HTTP POST payload
func (d *Dispatcher) DispatchEvent(ctx context.Context, event string, payload any) error {
	webhooks, err := d.queries.ListActiveWebhooksForEvent(ctx, []string{event})
	if err != nil {
		return fmt.Errorf("failed to list webhooks for event %s: %w", event, err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	for _, wh := range webhooks {
		go d.deliverWebhook(context.Background(), wh, event, payloadBytes)
	}

	return nil
}

func (d *Dispatcher) deliverWebhook(ctx context.Context, wh db.Webhooks, event string, payload []byte) {
	sig := ComputeSignature(wh.Secret, payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.Url, bytes.NewReader(payload))
	if err != nil {
		slog.Error("Failed to build webhook request", "webhook_id", types.UUIDToString(wh.ID), "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Helpdesk-Event", event)
	req.Header.Set("X-Helpdesk-Signature-256", sig)

	var customHeaders map[string]string
	if len(wh.Headers) > 0 && string(wh.Headers) != "{}" {
		if err := json.Unmarshal(wh.Headers, &customHeaders); err == nil {
			for k, v := range customHeaders {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := d.httpClient.Do(req)
	var respCode pgtype.Int4
	var deliveredAt pgtype.Timestamptz

	if err != nil {
		slog.Warn("Webhook delivery failed", "webhook_id", types.UUIDToString(wh.ID), "url", wh.Url, "error", err)
	} else {
		_ = resp.Body.Close()
		respCode = pgtype.Int4{Int32: int32(resp.StatusCode), Valid: true}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			deliveredAt = types.TimeToTimestamptz(time.Now().UTC())
		}
	}

	_ = d.queries.RecordWebhookDelivery(ctx, db.RecordWebhookDeliveryParams{
		ID:           types.NewUUIDv7(),
		WebhookID:    wh.ID,
		Event:        event,
		Payload:      payload,
		ResponseCode: respCode,
		Attempt:      1,
		DeliveredAt:  deliveredAt,
	})
}
