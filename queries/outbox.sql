-- name: InsertOutboxMessage :one
INSERT INTO outbox (
    id,
    aggregate,
    aggregate_id,
    event,
    payload
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ClaimUnpublishedOutboxBatch :many
SELECT * FROM outbox
WHERE published_at IS NULL AND retry_count < 10
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE outbox
SET published_at = (now() AT TIME ZONE 'utc')
WHERE id = $1;

-- name: MarkOutboxFailed :exec
UPDATE outbox
SET retry_count = retry_count + 1,
    last_error = $2
WHERE id = $1;
