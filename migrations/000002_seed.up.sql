-- Seed Initial Data

-- Seed Roles
INSERT INTO roles (id, key, label, description, is_system) VALUES
('018e0000-0000-7000-8000-000000000001', 'admin', 'Administrator', 'Full system access and configuration', TRUE),
('018e0000-0000-7000-8000-000000000002', 'manager', 'Support Manager', 'Manage queues, SLAs, teams, and reports', TRUE),
('018e0000-0000-7000-8000-000000000003', 'agent', 'Support Agent', 'Work on assigned and team tickets', TRUE),
('018e0000-0000-7000-8000-000000000004', 'restricted_agent', 'Restricted Agent', 'Work only on explicitly assigned tickets', TRUE),
('018e0000-0000-7000-8000-000000000005', 'contact', 'Customer Contact', 'Customer portal user access', TRUE)
ON CONFLICT (key) DO NOTHING;

-- Seed Permissions
INSERT INTO permissions (id, key, label, description) VALUES
('018e0000-0000-7000-8000-000000000010', 'tickets:read', 'View Tickets', 'View tickets in accessible queues'),
('018e0000-0000-7000-8000-000000000011', 'tickets:write', 'Modify Tickets', 'Update status, assignees, priorities, and fields'),
('018e0000-0000-7000-8000-000000000012', 'tickets:delete', 'Delete Tickets', 'Soft delete tickets'),
('018e0000-0000-7000-8000-000000000013', 'tickets:reply', 'Reply to Tickets', 'Send public replies to customers'),
('018e0000-0000-7000-8000-000000000014', 'tickets:internal_note', 'Add Internal Notes', 'Add internal notes to tickets'),
('018e0000-0000-7000-8000-000000000020', 'admin:manage_users', 'Manage Users & Agents', 'Create, update, and deactivate users and agents'),
('018e0000-0000-7000-8000-000000000021', 'admin:manage_teams', 'Manage Teams', 'Create and configure support teams'),
('018e0000-0000-7000-8000-000000000022', 'admin:manage_sla', 'Manage SLA Policies', 'Configure SLA policies, business hours, and targets'),
('018e0000-0000-7000-8000-000000000023', 'admin:manage_automations', 'Manage Automations', 'Create and edit assignment and automation rules'),
('018e0000-0000-7000-8000-000000000024', 'admin:manage_fields', 'Manage Custom Fields', 'Create and configure custom field definitions'),
('018e0000-0000-7000-8000-000000000025', 'admin:manage_mail', 'Manage Mail Accounts', 'Configure inbound and outbound mail channels'),
('018e0000-0000-7000-8000-000000000030', 'kb:manage', 'Manage Knowledge Base', 'Create, edit, and publish KB articles')
ON CONFLICT (key) DO NOTHING;

-- Grant Admin all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT '018e0000-0000-7000-8000-000000000001', id FROM permissions
ON CONFLICT DO NOTHING;

-- Seed Default Ticket Statuses
INSERT INTO ticket_statuses (id, key, label, category, pauses_sla, stops_sla, is_default, sort_order) VALUES
('018e0000-0000-7000-8000-000000000101', 'new', 'New', 'open', FALSE, FALSE, TRUE, 10),
('018e0000-0000-7000-8000-000000000102', 'open', 'Open', 'open', FALSE, FALSE, FALSE, 20),
('018e0000-0000-7000-8000-000000000103', 'pending', 'Pending Customer', 'pending', TRUE, FALSE, FALSE, 30),
('018e0000-0000-7000-8000-000000000104', 'on_hold', 'On Hold', 'paused', TRUE, FALSE, FALSE, 40),
('018e0000-0000-7000-8000-000000000105', 'resolved', 'Resolved', 'resolved', FALSE, TRUE, FALSE, 50),
('018e0000-0000-7000-8000-000000000106', 'closed', 'Closed', 'closed', FALSE, TRUE, FALSE, 60)
ON CONFLICT (key) DO NOTHING;

-- Seed Default Ticket Priorities
INSERT INTO ticket_priorities (id, key, label, weight, sort_order) VALUES
('018e0000-0000-7000-8000-000000000201', 'low', 'Low', 1, 10),
('018e0000-0000-7000-8000-000000000202', 'medium', 'Medium', 2, 20),
('018e0000-0000-7000-8000-000000000203', 'high', 'High', 3, 30),
('018e0000-0000-7000-8000-000000000204', 'urgent', 'Urgent', 4, 40)
ON CONFLICT (key) DO NOTHING;

-- Seed Default Ticket Types
INSERT INTO ticket_types (id, key, label, default_priority_id) VALUES
('018e0000-0000-7000-8000-000000000301', 'incident', 'Incident', '018e0000-0000-7000-8000-000000000202'),
('018e0000-0000-7000-8000-000000000302', 'question', 'Question', '018e0000-0000-7000-8000-000000000201'),
('018e0000-0000-7000-8000-000000000303', 'problem', 'Problem', '018e0000-0000-7000-8000-000000000203'),
('018e0000-0000-7000-8000-000000000304', 'feature_request', 'Feature Request', '018e0000-0000-7000-8000-000000000201')
ON CONFLICT (key) DO NOTHING;

-- Seed Default Business Hours (Mon-Fri 09:00 - 17:00 UTC)
INSERT INTO business_hours (id, name, timezone) VALUES
('018e0000-0000-7000-8000-000000000401', 'Standard Support Hours (9x5)', 'UTC')
ON CONFLICT DO NOTHING;

INSERT INTO business_hour_slots (business_hours_id, weekday, start_time, end_time) VALUES
('018e0000-0000-7000-8000-000000000401', 1, '09:00:00', '17:00:00'),
('018e0000-0000-7000-8000-000000000401', 2, '09:00:00', '17:00:00'),
('018e0000-0000-7000-8000-000000000401', 3, '09:00:00', '17:00:00'),
('018e0000-0000-7000-8000-000000000401', 4, '09:00:00', '17:00:00'),
('018e0000-0000-7000-8000-000000000401', 5, '09:00:00', '17:00:00')
ON CONFLICT DO NOTHING;

-- Seed Default SLA Policy & Targets
INSERT INTO sla_policies (id, name, is_active, priority_order, business_hours_id, apply_to) VALUES
('018e0000-0000-7000-8000-000000000501', 'Default General SLA', TRUE, 100, '018e0000-0000-7000-8000-000000000401', 'all')
ON CONFLICT DO NOTHING;

INSERT INTO sla_targets (policy_id, priority_id, first_response_minutes, resolution_minutes) VALUES
('018e0000-0000-7000-8000-000000000501', '018e0000-0000-7000-8000-000000000204', 60, 240),    -- Urgent: 1h / 4h
('018e0000-0000-7000-8000-000000000501', '018e0000-0000-7000-8000-000000000203', 120, 480),   -- High: 2h / 8h
('018e0000-0000-7000-8000-000000000501', '018e0000-0000-7000-8000-000000000202', 480, 1440),  -- Med: 8h / 24h
('018e0000-0000-7000-8000-000000000501', '018e0000-0000-7000-8000-000000000201', 1440, 2880)  -- Low: 24h / 48h
ON CONFLICT (policy_id, priority_id) DO NOTHING;

-- Seed Default Users:
-- 1. admin / admin (admin@helpdesk.local)
INSERT INTO users (id, email, password_hash, full_name, timezone, locale, is_active) VALUES
('018e0000-0000-7000-8000-000000000099', 'admin@helpdesk.local', '$2a$12$4eI8u5aM6j15NlS6gKqyieYg5zE0o7l.cK4gC8lV1KkQeZq7nJp5W', 'System Administrator', 'UTC', 'en', TRUE)
ON CONFLICT (email) DO NOTHING;

INSERT INTO agents (id, user_id, display_name, signature_html, is_available, max_concurrent_tickets, skills) VALUES
('018e0000-0000-7000-8000-000000000088', '018e0000-0000-7000-8000-000000000099', 'System Admin', '<p>Best regards,<br/><b>System Administrator</b></p>', TRUE, 20, ARRAY['general', 'technical'])
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id) VALUES
('018e0000-0000-7000-8000-000000000099', '018e0000-0000-7000-8000-000000000001')
ON CONFLICT DO NOTHING;

-- 2. support / support (support@helpdesk.local)
INSERT INTO users (id, email, password_hash, full_name, timezone, locale, is_active) VALUES
('018e0000-0000-7000-8000-000000000098', 'support@helpdesk.local', '$2a$12$4eI8u5aM6j15NlS6gKqyieYg5zE0o7l.cK4gC8lV1KkQeZq7nJp5W', 'Support Agent', 'UTC', 'en', TRUE)
ON CONFLICT (email) DO NOTHING;

INSERT INTO agents (id, user_id, display_name, signature_html, is_available, max_concurrent_tickets, skills) VALUES
('018e0000-0000-7000-8000-000000000087', '018e0000-0000-7000-8000-000000000098', 'Support Agent', '<p>Best regards,<br/><b>Support Team</b></p>', TRUE, 20, ARRAY['general', 'technical'])
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id) VALUES
('018e0000-0000-7000-8000-000000000098', '018e0000-0000-7000-8000-000000000003')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id) VALUES
('018e0000-0000-7000-8000-000000000003', '018e0000-0000-7000-8000-000000000010'),
('018e0000-0000-7000-8000-000000000003', '018e0000-0000-7000-8000-000000000011'),
('018e0000-0000-7000-8000-000000000003', '018e0000-0000-7000-8000-000000000013'),
('018e0000-0000-7000-8000-000000000003', '018e0000-0000-7000-8000-000000000014')
ON CONFLICT DO NOTHING;

-- 3. customer / customer (customer@helpdesk.local)
INSERT INTO users (id, email, password_hash, full_name, timezone, locale, is_active) VALUES
('018e0000-0000-7000-8000-000000000097', 'customer@helpdesk.local', '$2a$12$4eI8u5aM6j15NlS6gKqyieYg5zE0o7l.cK4gC8lV1KkQeZq7nJp5W', 'Customer User', 'UTC', 'en', TRUE)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_roles (user_id, role_id) VALUES
('018e0000-0000-7000-8000-000000000097', '018e0000-0000-7000-8000-000000000005')
ON CONFLICT DO NOTHING;

INSERT INTO contacts (id, primary_email, full_name, locale, timezone, is_verified, portal_user_id) VALUES
('018e0000-0000-7000-8000-000000000077', 'customer@helpdesk.local', 'Customer User', 'en', 'UTC', TRUE, '018e0000-0000-7000-8000-000000000097')
ON CONFLICT (primary_email) DO NOTHING;

-- Seed KB Space and Default Category
INSERT INTO kb_spaces (id, name, slug, visibility) VALUES
('018e0000-0000-7000-8000-000000000601', 'Help Center', 'help-center', 'public')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO kb_categories (id, space_id, name, slug, sort_order) VALUES
('018e0000-0000-7000-8000-000000000610', '018e0000-0000-7000-8000-000000000601', 'Getting Started', 'getting-started', 10),
('018e0000-0000-7000-8000-000000000611', '018e0000-0000-7000-8000-000000000601', 'Account & Billing', 'account-billing', 20),
('018e0000-0000-7000-8000-000000000612', '018e0000-0000-7000-8000-000000000601', 'Troubleshooting', 'troubleshooting', 30)
ON CONFLICT (space_id, slug) DO NOTHING;
