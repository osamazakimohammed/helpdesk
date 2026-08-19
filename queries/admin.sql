-- name: ListFieldDefinitions :many
SELECT * FROM field_definitions WHERE entity = $1 AND deleted_at IS NULL ORDER BY sort_order ASC;

-- name: ListAllFieldDefinitions :many
SELECT * FROM field_definitions WHERE deleted_at IS NULL ORDER BY entity, sort_order ASC;

-- name: ListAssignmentRules :many
SELECT * FROM assignment_rules WHERE deleted_at IS NULL ORDER BY priority_order ASC;

-- name: ListActiveAssignmentRules :many
SELECT * FROM assignment_rules 
WHERE is_active = TRUE AND trigger = $1 AND deleted_at IS NULL 
ORDER BY priority_order ASC;

-- name: UpdateAssignmentRuleLastAgent :exec
UPDATE assignment_rules SET last_assigned_agent_id = $2 WHERE id = $1;

-- name: ListAutomationRules :many
SELECT * FROM automation_rules WHERE deleted_at IS NULL ORDER BY priority_order ASC;

-- name: ListActiveAutomationRulesByTrigger :many
SELECT * FROM automation_rules 
WHERE is_active = TRUE AND trigger = $1 AND deleted_at IS NULL 
ORDER BY priority_order ASC;

-- name: RecordAutomationRun :exec
INSERT INTO automation_runs (id, rule_id, ticket_id, status, error, actions_applied)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListActiveWebhooksForEvent :many
SELECT * FROM webhooks 
WHERE is_active = TRUE AND $1 = ANY(events) AND deleted_at IS NULL;

-- name: RecordWebhookDelivery :exec
INSERT INTO webhook_deliveries (id, webhook_id, event, payload, response_code, attempt, next_retry_at, delivered_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListMailAccounts :many
SELECT * FROM mail_accounts WHERE deleted_at IS NULL ORDER BY label ASC;

-- name: ListInboundMailAccounts :many
SELECT * FROM mail_accounts 
WHERE direction IN ('inbound', 'both') AND deleted_at IS NULL;

-- name: UpdateMailAccountPolled :exec
UPDATE mail_accounts 
SET last_polled_at = (now() AT TIME ZONE 'utc'), last_uid = $2 
WHERE id = $1;

-- name: ListMailFilters :many
SELECT * FROM mail_filters WHERE mail_account_id = $1 AND deleted_at IS NULL;

-- name: IsEmailSuppressed :one
SELECT EXISTS(SELECT 1 FROM mail_suppressions WHERE email = $1 AND deleted_at IS NULL) AS is_suppressed;

-- name: AddMailSuppression :exec
INSERT INTO mail_suppressions (id, email, reason)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO NOTHING;

-- name: ListReplyTemplates :many
SELECT * FROM reply_templates 
WHERE is_active = TRUE 
  AND deleted_at IS NULL 
  AND (scope = 'global' OR (scope = 'team' AND team_id = sqlc.narg('team_id')::uuid) OR (scope = 'personal' AND owner_agent_id = sqlc.narg('agent_id')::uuid))
ORDER BY usage_count DESC, name ASC;

-- name: IncrementReplyTemplateUsage :exec
UPDATE reply_templates SET usage_count = usage_count + 1 WHERE id = $1;

-- name: InsertAuditLog :exec
INSERT INTO audit_log (id, actor_id, actor_type, entity, entity_id, action, before, after, ip, user_agent, at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now() AT TIME ZONE 'utc');

-- name: ListAuditLogsKeyset :many
SELECT * FROM audit_log
WHERE ($1::uuid IS NULL OR entity_id = $1)
  AND ($2::text IS NULL OR entity = $2)
  AND (
    sqlc.narg('after_at')::timestamptz IS NULL
    OR (at, id) < (sqlc.narg('after_at'), sqlc.narg('after_id')::uuid)
  )
ORDER BY at DESC, id DESC
LIMIT $3;
