-- name: ListTicketEventsKeyset :many
-- Query 2 of 2 for Ticket Detail: Keyset paginated timeline events with attachments and mentions
SELECT 
    e.id,
    e.ticket_id,
    e.kind,
    e.visibility,
    e.author_type,
    e.author_id,
    CASE 
        WHEN e.author_type = 'agent' THEN (SELECT display_name FROM agents WHERE id = e.author_id)
        WHEN e.author_type = 'contact' THEN (SELECT full_name FROM contacts WHERE id = e.author_id)
        ELSE 'System'
    END AS author_name,
    e.body_html,
    e.body_text,
    e.metadata,
    e.email_message_id,
    e.email_in_reply_to,
    e.email_references,
    e.delivery_status,
    e.occurred_at,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'id', att.id,
            'filename', att.filename,
            'mime_type', att.mime_type,
            'size_bytes', att.size_bytes,
            'storage_key', att.storage_key,
            'is_inline', att.is_inline,
            'content_id', att.content_id
        ))
        FROM attachments att
        WHERE att.event_id = e.id),
        '[]'::json
    ) AS attachments_json,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'agent_id', m.agent_id,
            'agent_name', ag.display_name,
            'read_at', m.read_at
        ))
        FROM mentions m
        JOIN agents ag ON ag.id = m.agent_id
        WHERE m.event_id = e.id),
        '[]'::json
    ) AS mentions_json
FROM ticket_events e
WHERE e.ticket_id = $1
  AND (sqlc.narg('visibility')::text IS NULL OR sqlc.narg('visibility')::text = '' OR e.visibility = sqlc.narg('visibility'))
  AND (
    sqlc.narg('after_occurred_at')::timestamptz IS NULL
    OR (e.occurred_at, e.id) < (sqlc.narg('after_occurred_at'), sqlc.narg('after_id')::uuid)
  )
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT $2;

-- name: CreateTicketEvent :one
INSERT INTO ticket_events (
    id,
    ticket_id,
    kind,
    visibility,
    author_type,
    author_id,
    body_html,
    body_text,
    metadata,
    email_message_id,
    email_in_reply_to,
    email_references,
    delivery_status,
    occurred_at,
    created_by
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15
) RETURNING *;

-- name: CreateAttachment :one
INSERT INTO attachments (
    id,
    ticket_id,
    event_id,
    filename,
    mime_type,
    size_bytes,
    storage_key,
    checksum_sha256,
    is_inline,
    content_id,
    scan_status,
    created_by
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12
) RETURNING *;

-- name: GetAttachmentsByTicket :many
SELECT * FROM attachments WHERE ticket_id = $1 AND deleted_at IS NULL;

-- name: CreateMention :exec
INSERT INTO mentions (id, event_id, agent_id) VALUES ($1, $2, $3);

-- name: MarkMentionRead :exec
UPDATE mentions SET read_at = (now() AT TIME ZONE 'utc') WHERE event_id = $1 AND agent_id = $2;

-- name: GetEventByEmailMessageID :one
SELECT * FROM ticket_events WHERE email_message_id = $1 LIMIT 1;
