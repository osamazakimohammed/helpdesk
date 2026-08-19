-- name: ListTicketsKeyset :many
-- 1 Query Ticket List with Keyset Pagination and complete aggregated details
SELECT 
    t.id,
    t.reference_no,
    t.subject,
    t.description_text,
    t.status_id,
    ts.key AS status_key,
    ts.label AS status_label,
    t.status_category,
    t.priority_id,
    tp.key AS priority_key,
    tp.label AS priority_label,
    t.priority_weight,
    t.type_id,
    tt.key AS type_key,
    tt.label AS type_label,
    t.contact_id,
    c.full_name AS contact_name,
    c.primary_email AS contact_email,
    t.organization_id,
    org.name AS organization_name,
    t.assigned_agent_id,
    ag.display_name AS assigned_agent_name,
    t.assigned_team_id,
    tm.name AS assigned_team_name,
    t.source,
    t.first_response_due_at,
    t.first_responded_at,
    t.resolution_due_at,
    t.resolved_at,
    t.closed_at,
    t.first_response_breached,
    t.resolution_breached,
    t.reopen_count,
    t.last_customer_activity_at,
    t.last_agent_activity_at,
    t.custom_data,
    t.created_at,
    t.updated_at,
    COALESCE(
        (SELECT json_agg(json_build_object('id', tg.id, 'key', tg.key, 'label', tg.label))
         FROM ticket_tags tt_tag
         JOIN tags tg ON tg.id = tt_tag.tag_id
         WHERE tt_tag.ticket_id = t.id),
        '[]'::json
    ) AS tags_json
FROM tickets t
JOIN ticket_statuses ts ON ts.id = t.status_id
JOIN ticket_priorities tp ON tp.id = t.priority_id
LEFT JOIN ticket_types tt ON tt.id = t.type_id
JOIN contacts c ON c.id = t.contact_id
LEFT JOIN organizations org ON org.id = t.organization_id
LEFT JOIN agents ag ON ag.id = t.assigned_agent_id
LEFT JOIN teams tm ON tm.id = t.assigned_team_id
WHERE t.deleted_at IS NULL
  AND (
    sqlc.narg('status_category')::text IS NULL 
    OR (sqlc.narg('status_category')::text = 'resolved' AND t.status_category IN ('resolved', 'closed'))
    OR (sqlc.narg('status_category')::text = 'open' AND t.status_category IN ('open', 'new'))
    OR (sqlc.narg('status_category')::text = 'pending' AND t.status_category IN ('pending', 'paused'))
    OR t.status_category = sqlc.narg('status_category')
  )
  AND (sqlc.narg('assigned_agent_id')::uuid IS NULL OR t.assigned_agent_id = sqlc.narg('assigned_agent_id'))
  AND (sqlc.narg('assigned_team_id')::uuid IS NULL OR t.assigned_team_id = sqlc.narg('assigned_team_id'))
  AND (sqlc.narg('contact_id')::uuid IS NULL OR t.contact_id = sqlc.narg('contact_id'))
  AND (sqlc.narg('search_query')::text IS NULL OR t.search_vector @@ websearch_to_tsquery('english', sqlc.narg('search_query')))
  AND (
    sqlc.narg('after_updated_at')::timestamptz IS NULL 
    OR (t.updated_at, t.id) < (sqlc.narg('after_updated_at'), sqlc.narg('after_id')::uuid)
  )
ORDER BY t.updated_at DESC, t.id DESC
LIMIT $1;

-- name: GetTicketDetail :one
-- Query 1 of 2 for Ticket Detail: Full Ticket metadata with associations
SELECT 
    t.id,
    t.reference_no,
    t.subject,
    t.description_html,
    t.description_text,
    t.status_id,
    ts.key AS status_key,
    ts.label AS status_label,
    ts.category AS status_category,
    ts.pauses_sla,
    ts.stops_sla,
    t.priority_id,
    tp.key AS priority_key,
    tp.label AS priority_label,
    t.priority_weight,
    t.type_id,
    tt.key AS type_key,
    tt.label AS type_label,
    t.contact_id,
    c.full_name AS contact_name,
    c.primary_email AS contact_email,
    c.phone AS contact_phone,
    c.locale AS contact_locale,
    t.organization_id,
    org.name AS organization_name,
    org.domain AS organization_domain,
    t.assigned_agent_id,
    ag.display_name AS assigned_agent_name,
    ag.user_id AS assigned_agent_user_id,
    t.assigned_team_id,
    tm.name AS assigned_team_name,
    t.source,
    t.sla_policy_id,
    sp.name AS sla_policy_name,
    t.first_response_due_at,
    t.first_responded_at,
    t.resolution_due_at,
    t.resolved_at,
    t.closed_at,
    t.sla_paused_at,
    t.sla_paused_ms,
    t.first_response_breached,
    t.resolution_breached,
    t.reopen_count,
    t.merged_into_id,
    t.parent_ticket_id,
    t.last_customer_activity_at,
    t.last_agent_activity_at,
    t.feedback_rating,
    t.feedback_comment,
    t.custom_data,
    t.created_at,
    t.updated_at,
    COALESCE(
        (SELECT json_agg(json_build_object('id', tg.id, 'key', tg.key, 'label', tg.label))
         FROM ticket_tags tt_tag
         JOIN tags tg ON tg.id = tt_tag.tag_id
         WHERE tt_tag.ticket_id = t.id),
        '[]'::json
    ) AS tags_json,
    COALESCE(
        (SELECT json_agg(json_build_object('id', u.id, 'name', u.full_name, 'email', u.email))
         FROM ticket_watchers tw
         JOIN users u ON u.id = tw.user_id
         WHERE tw.ticket_id = t.id),
        '[]'::json
    ) AS watchers_json,
    COALESCE(
        (SELECT json_agg(json_build_object('from_ticket_id', tl.from_ticket_id, 'to_ticket_id', tl.to_ticket_id, 'relation', tl.relation, 'to_reference_no', t2.reference_no, 'to_subject', t2.subject))
         FROM ticket_links tl
         JOIN tickets t2 ON t2.id = tl.to_ticket_id
         WHERE tl.from_ticket_id = t.id),
        '[]'::json
    ) AS links_json
FROM tickets t
JOIN ticket_statuses ts ON ts.id = t.status_id
JOIN ticket_priorities tp ON tp.id = t.priority_id
LEFT JOIN ticket_types tt ON tt.id = t.type_id
JOIN contacts c ON c.id = t.contact_id
LEFT JOIN organizations org ON org.id = t.organization_id
LEFT JOIN agents ag ON ag.id = t.assigned_agent_id
LEFT JOIN teams tm ON tm.id = t.assigned_team_id
LEFT JOIN sla_policies sp ON sp.id = t.sla_policy_id
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- name: GetTicketByReferenceNo :one
SELECT 
    id, reference_no, subject, description_html, description_text, status_id, status_category,
    priority_id, priority_weight, type_id, contact_id, organization_id, assigned_agent_id, assigned_team_id,
    source, sla_policy_id, first_response_due_at, first_responded_at, resolution_due_at, resolved_at, closed_at,
    sla_paused_at, sla_paused_ms, first_response_breached, resolution_breached, reopen_count, merged_into_id,
    parent_ticket_id, last_customer_activity_at, last_agent_activity_at, feedback_rating, feedback_comment,
    custom_data, created_at, updated_at, created_by, updated_by, deleted_at
FROM tickets WHERE reference_no = $1 AND deleted_at IS NULL;

-- name: CreateTicket :one
INSERT INTO tickets (
    id,
    reference_no,
    subject,
    description_html,
    description_text,
    status_id,
    status_category,
    priority_id,
    priority_weight,
    type_id,
    contact_id,
    organization_id,
    assigned_agent_id,
    assigned_team_id,
    source,
    sla_policy_id,
    first_response_due_at,
    resolution_due_at,
    custom_data,
    created_by
) VALUES (
    $1,
    'TCK-' || nextval('ticket_reference_seq')::text,
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
    $15,
    $16,
    $17,
    $18,
    $19
) RETURNING 
    id, reference_no, subject, description_html, description_text, status_id, status_category,
    priority_id, priority_weight, type_id, contact_id, organization_id, assigned_agent_id, assigned_team_id,
    source, sla_policy_id, first_response_due_at, first_responded_at, resolution_due_at, resolved_at, closed_at,
    sla_paused_at, sla_paused_ms, first_response_breached, resolution_breached, reopen_count, merged_into_id,
    parent_ticket_id, last_customer_activity_at, last_agent_activity_at, feedback_rating, feedback_comment,
    custom_data, created_at, updated_at, created_by, updated_by, deleted_at;

-- name: UpdateTicketFields :one
UPDATE tickets
SET 
    subject = COALESCE(sqlc.narg('subject'), subject),
    description_html = COALESCE(sqlc.narg('description_html'), description_html),
    description_text = COALESCE(sqlc.narg('description_text'), description_text),
    status_id = COALESCE(sqlc.narg('status_id'), status_id),
    status_category = COALESCE(sqlc.narg('status_category'), status_category),
    priority_id = COALESCE(sqlc.narg('priority_id'), priority_id),
    priority_weight = COALESCE(sqlc.narg('priority_weight'), priority_weight),
    type_id = COALESCE(sqlc.narg('type_id'), type_id),
    assigned_agent_id = sqlc.narg('assigned_agent_id'),
    assigned_team_id = sqlc.narg('assigned_team_id'),
    sla_policy_id = COALESCE(sqlc.narg('sla_policy_id'), sla_policy_id),
    first_response_due_at = sqlc.narg('first_response_due_at'),
    first_responded_at = COALESCE(sqlc.narg('first_responded_at'), first_responded_at),
    resolution_due_at = sqlc.narg('resolution_due_at'),
    resolved_at = sqlc.narg('resolved_at'),
    closed_at = sqlc.narg('closed_at'),
    sla_paused_at = sqlc.narg('sla_paused_at'),
    sla_paused_ms = COALESCE(sqlc.narg('sla_paused_ms'), sla_paused_ms),
    first_response_breached = COALESCE(sqlc.narg('first_response_breached'), first_response_breached),
    resolution_breached = COALESCE(sqlc.narg('resolution_breached'), resolution_breached),
    reopen_count = COALESCE(sqlc.narg('reopen_count'), reopen_count),
    last_customer_activity_at = COALESCE(sqlc.narg('last_customer_activity_at'), last_customer_activity_at),
    last_agent_activity_at = COALESCE(sqlc.narg('last_agent_activity_at'), last_agent_activity_at),
    feedback_rating = COALESCE(sqlc.narg('feedback_rating'), feedback_rating),
    feedback_comment = COALESCE(sqlc.narg('feedback_comment'), feedback_comment),
    custom_data = COALESCE(sqlc.narg('custom_data'), custom_data),
    updated_at = (now() AT TIME ZONE 'utc'),
    updated_by = sqlc.narg('updated_by')
WHERE id = $1 AND deleted_at IS NULL
RETURNING 
    id, reference_no, subject, description_html, description_text, status_id, status_category,
    priority_id, priority_weight, type_id, contact_id, organization_id, assigned_agent_id, assigned_team_id,
    source, sla_policy_id, first_response_due_at, first_responded_at, resolution_due_at, resolved_at, closed_at,
    sla_paused_at, sla_paused_ms, first_response_breached, resolution_breached, reopen_count, merged_into_id,
    parent_ticket_id, last_customer_activity_at, last_agent_activity_at, feedback_rating, feedback_comment,
    custom_data, created_at, updated_at, created_by, updated_by, deleted_at;

-- name: SoftDeleteTicket :exec
UPDATE tickets 
SET deleted_at = (now() AT TIME ZONE 'utc'), updated_by = $2
WHERE id = $1;

-- name: AddTicketTag :exec
INSERT INTO ticket_tags (ticket_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveTicketTag :exec
DELETE FROM ticket_tags WHERE ticket_id = $1 AND tag_id = $2;

-- name: AddTicketWatcher :exec
INSERT INTO ticket_watchers (ticket_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveTicketWatcher :exec
DELETE FROM ticket_watchers WHERE ticket_id = $1 AND user_id = $2;

-- name: LinkTickets :exec
INSERT INTO ticket_links (from_ticket_id, to_ticket_id, relation) 
VALUES ($1, $2, $3) 
ON CONFLICT (from_ticket_id, to_ticket_id, relation) DO NOTHING;

-- name: UnlinkTickets :exec
DELETE FROM ticket_links WHERE from_ticket_id = $1 AND to_ticket_id = $2 AND relation = $3;
