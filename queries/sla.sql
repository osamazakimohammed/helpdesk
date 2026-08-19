-- name: ListActiveSLAPolicies :many
SELECT 
    sp.id,
    sp.name,
    sp.priority_order,
    sp.match_conditions,
    sp.business_hours_id,
    sp.holiday_calendar_id,
    sp.apply_to,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'priority_id', st.priority_id,
            'first_response_minutes', st.first_response_minutes,
            'resolution_minutes', st.resolution_minutes,
            'next_response_minutes', st.next_response_minutes
        ))
        FROM sla_targets st
        WHERE st.policy_id = sp.id),
        '[]'::json
    ) AS targets_json
FROM sla_policies sp
WHERE sp.is_active = TRUE AND sp.deleted_at IS NULL
ORDER BY sp.priority_order ASC;

-- name: GetBusinessHoursWithSlots :one
SELECT 
    bh.id,
    bh.name,
    bh.timezone,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'weekday', bhs.weekday,
            'start_time', bhs.start_time::text,
            'end_time', bhs.end_time::text
        ) ORDER BY bhs.weekday, bhs.start_time)
        FROM business_hour_slots bhs
        WHERE bhs.business_hours_id = bh.id),
        '[]'::json
    ) AS slots_json
FROM business_hours bh
WHERE bh.id = $1 AND bh.deleted_at IS NULL;

-- name: GetHolidayCalendarWithHolidays :one
SELECT 
    hc.id,
    hc.name,
    hc.timezone,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'date', h.date::text,
            'label', h.label,
            'is_recurring_annual', h.is_recurring_annual
        ))
        FROM holidays h
        WHERE h.calendar_id = hc.id),
        '[]'::json
    ) AS holidays_json
FROM holiday_calendars hc
WHERE hc.id = $1 AND hc.deleted_at IS NULL;

-- name: GetSLAPolicyTargets :many
SELECT * FROM sla_targets WHERE policy_id = $1 AND deleted_at IS NULL;

-- name: GetSLAPolicyEscalations :many
SELECT * FROM sla_escalations WHERE policy_id = $1 AND deleted_at IS NULL;

-- name: ScanBreachingFirstResponseTickets :many
-- Uses the partial index: tickets(first_response_due_at) WHERE first_responded_at IS NULL AND sla_paused_at IS NULL AND deleted_at IS NULL
SELECT id, reference_no, sla_policy_id, priority_id, first_response_due_at, assigned_agent_id, assigned_team_id
FROM tickets
WHERE first_response_due_at <= $1
  AND first_responded_at IS NULL
  AND sla_paused_at IS NULL
  AND first_response_breached = FALSE
  AND deleted_at IS NULL
LIMIT $2;

-- name: ScanBreachingResolutionTickets :many
-- Uses the partial index: tickets(resolution_due_at) WHERE resolved_at IS NULL AND sla_paused_at IS NULL AND deleted_at IS NULL
SELECT id, reference_no, sla_policy_id, priority_id, resolution_due_at, assigned_agent_id, assigned_team_id
FROM tickets
WHERE resolution_due_at <= $1
  AND resolved_at IS NULL
  AND sla_paused_at IS NULL
  AND resolution_breached = FALSE
  AND deleted_at IS NULL
LIMIT $2;

-- name: MarkFirstResponseBreached :exec
UPDATE tickets SET first_response_breached = TRUE, updated_at = (now() AT TIME ZONE 'utc') WHERE id = $1;

-- name: MarkResolutionBreached :exec
UPDATE tickets SET resolution_breached = TRUE, updated_at = (now() AT TIME ZONE 'utc') WHERE id = $1;
