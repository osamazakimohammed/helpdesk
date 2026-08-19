-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAgentByUserID :one
SELECT * FROM agents WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetContactByEmail :one
SELECT * FROM contacts WHERE primary_email = $1 AND deleted_at IS NULL;

-- name: GetContactByID :one
SELECT * FROM contacts WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserRoleKeys :many
SELECT r.key
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1;

-- name: GetUserPermissions :many
SELECT DISTINCT p.key
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;

-- name: ListAgents :many
SELECT 
    ag.id,
    ag.user_id,
    ag.display_name,
    ag.signature_html,
    ag.is_available,
    ag.max_concurrent_tickets,
    ag.skills,
    u.email,
    (SELECT count(*) FROM tickets t WHERE t.assigned_agent_id = ag.id AND t.deleted_at IS NULL AND t.status_category IN ('open', 'pending')) AS open_ticket_count
FROM agents ag
JOIN users u ON u.id = ag.user_id
WHERE ag.deleted_at IS NULL AND u.deleted_at IS NULL
ORDER BY ag.display_name ASC;

-- name: ListTeams :many
SELECT 
    tm.id,
    tm.name,
    tm.slug,
    tm.lead_agent_id,
    tm.inbox_email,
    tm.assignment_strategy,
    tm.restrict_visibility,
    (SELECT count(*) FROM team_members tmb WHERE tmb.team_id = tm.id AND tmb.deleted_at IS NULL) AS member_count,
    (SELECT count(*) FROM tickets t WHERE t.assigned_team_id = tm.id AND t.deleted_at IS NULL AND t.status_category IN ('open', 'pending')) AS open_ticket_count
FROM teams tm
WHERE tm.deleted_at IS NULL
ORDER BY tm.name ASC;

-- name: CreateContact :one
INSERT INTO contacts (
    id,
    organization_id,
    primary_email,
    full_name,
    phone,
    locale,
    timezone,
    is_verified,
    portal_user_id,
    custom_data,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: CreateAgent :one
INSERT INTO agents (
    id,
    user_id,
    display_name,
    signature_html,
    is_available,
    max_concurrent_tickets,
    skills
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetOrCreateOrganizationByDomain :one
INSERT INTO organizations (
    id,
    name,
    slug,
    domain
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (slug) DO UPDATE SET updated_at = (now() AT TIME ZONE 'utc')
RETURNING *;

-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    full_name,
    timezone,
    locale,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: GetRoleByKey :one
SELECT * FROM roles WHERE key = $1 LIMIT 1;
