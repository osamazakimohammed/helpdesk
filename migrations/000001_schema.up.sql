-- Helpdesk PostgreSQL 17 Core Schema Migration

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- UUIDv7 Generator Function
CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS uuid AS $$
DECLARE
  v_time numeric;
  v_millis bigint;
  v_hex text;
BEGIN
  v_time := EXTRACT(EPOCH FROM clock_timestamp());
  v_millis := trunc(v_time * 1000);
  v_hex := lpad(to_hex(v_millis), 12, '0') 
           || '7' 
           || substr(md5(random()::text), 1, 3) 
           || '8' 
           || substr(md5(random()::text), 4, 3) 
           || substr(md5(random()::text), 7, 12);
  RETURN v_hex::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Sequence for ticket reference numbers (e.g. TCK-10001)
CREATE SEQUENCE IF NOT EXISTS ticket_reference_seq START WITH 10001 INCREMENT BY 1;

-- 1. Organizations
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    domain VARCHAR(255),
    default_timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 2. Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    locale VARCHAR(10) NOT NULL DEFAULT 'en',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    mfa_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 3. Agents
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL,
    signature_html TEXT NOT NULL DEFAULT '',
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    max_concurrent_tickets INT NOT NULL DEFAULT 10,
    skills TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 4. Teams
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    lead_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    inbox_email VARCHAR(255),
    assignment_strategy VARCHAR(30) NOT NULL DEFAULT 'round_robin' CHECK (assignment_strategy IN ('round_robin', 'least_open', 'least_weighted', 'skill_match', 'manual')),
    restrict_visibility BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 5. Team Members
CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'lead')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(team_id, agent_id)
);

-- 6. Contacts
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    primary_email VARCHAR(255) NOT NULL UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    locale VARCHAR(10) NOT NULL DEFAULT 'en',
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    portal_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    custom_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 7. Contact Additional Emails
CREATE TABLE IF NOT EXISTS contact_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 8. Ticket Statuses
CREATE TABLE IF NOT EXISTS ticket_statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(100) NOT NULL,
    category VARCHAR(20) NOT NULL CHECK (category IN ('open', 'pending', 'paused', 'resolved', 'closed')),
    pauses_sla BOOLEAN NOT NULL DEFAULT FALSE,
    stops_sla BOOLEAN NOT NULL DEFAULT FALSE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 9. Ticket Priorities
CREATE TABLE IF NOT EXISTS ticket_priorities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(100) NOT NULL,
    weight INT NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 10. Ticket Types
CREATE TABLE IF NOT EXISTS ticket_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(100) NOT NULL,
    default_priority_id UUID REFERENCES ticket_priorities(id) ON DELETE SET NULL,
    default_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 11. Tags
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 12. Business Hours & Holiday Calendars
CREATE TABLE IF NOT EXISTS business_hours (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS business_hour_slots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    business_hours_id UUID NOT NULL REFERENCES business_hours(id) ON DELETE CASCADE,
    weekday INT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS holiday_calendars (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    calendar_id UUID NOT NULL REFERENCES holiday_calendars(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    label VARCHAR(255) NOT NULL,
    is_recurring_annual BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 13. SLA Policies, Targets & Escalations
CREATE TABLE IF NOT EXISTS sla_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority_order INT NOT NULL DEFAULT 100,
    match_conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    business_hours_id UUID REFERENCES business_hours(id) ON DELETE SET NULL,
    holiday_calendar_id UUID REFERENCES holiday_calendars(id) ON DELETE SET NULL,
    apply_to VARCHAR(20) NOT NULL DEFAULT 'all' CHECK (apply_to IN ('all', 'conditional')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sla_targets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    policy_id UUID NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    priority_id UUID NOT NULL REFERENCES ticket_priorities(id) ON DELETE CASCADE,
    first_response_minutes INT NOT NULL,
    resolution_minutes INT NOT NULL,
    next_response_minutes INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(policy_id, priority_id)
);

CREATE TABLE IF NOT EXISTS sla_escalations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    policy_id UUID NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    target VARCHAR(30) NOT NULL CHECK (target IN ('first_response', 'resolution', 'next_response')),
    trigger_at_pct INT,
    offset_minutes INT,
    actions JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 14. Tickets
CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    reference_no VARCHAR(32) NOT NULL UNIQUE,
    subject TEXT NOT NULL,
    description_html TEXT NOT NULL,
    description_text TEXT NOT NULL,
    status_id UUID NOT NULL REFERENCES ticket_statuses(id),
    status_category VARCHAR(20) NOT NULL,
    priority_id UUID NOT NULL REFERENCES ticket_priorities(id),
    priority_weight INT NOT NULL DEFAULT 1,
    type_id UUID REFERENCES ticket_types(id) ON DELETE SET NULL,
    contact_id UUID NOT NULL REFERENCES contacts(id),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    assigned_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    assigned_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'form' CHECK (source IN ('email','portal','form','api','agent','phone','chat')),
    sla_policy_id UUID REFERENCES sla_policies(id) ON DELETE SET NULL,
    first_response_due_at TIMESTAMPTZ,
    first_responded_at TIMESTAMPTZ,
    resolution_due_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    sla_paused_at TIMESTAMPTZ,
    sla_paused_ms BIGINT NOT NULL DEFAULT 0,
    first_response_breached BOOLEAN NOT NULL DEFAULT FALSE,
    resolution_breached BOOLEAN NOT NULL DEFAULT FALSE,
    reopen_count INT NOT NULL DEFAULT 0,
    merged_into_id UUID REFERENCES tickets(id) ON DELETE SET NULL,
    parent_ticket_id UUID REFERENCES tickets(id) ON DELETE SET NULL,
    last_customer_activity_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    last_agent_activity_at TIMESTAMPTZ,
    feedback_rating INT CHECK (feedback_rating BETWEEN 1 AND 5),
    feedback_comment TEXT,
    custom_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    search_vector TSVECTOR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 15. Ticket Tags, Watchers, Links
CREATE TABLE IF NOT EXISTS ticket_tags (
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    PRIMARY KEY (ticket_id, tag_id)
);

CREATE TABLE IF NOT EXISTS ticket_watchers (
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    PRIMARY KEY (ticket_id, user_id)
);

CREATE TABLE IF NOT EXISTS ticket_links (
    from_ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    to_ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    relation VARCHAR(20) NOT NULL CHECK (relation IN ('duplicate', 'related', 'blocks', 'child')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    PRIMARY KEY (from_ticket_id, to_ticket_id, relation)
);

-- 16. Ticket Events (Timeline)
CREATE TABLE IF NOT EXISTS ticket_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    kind VARCHAR(30) NOT NULL CHECK (kind IN (
        'inbound_email','outbound_email','portal_reply','internal_note',
        'status_change','priority_change','assignment_change','team_change',
        'sla_event','tag_change','merge','attachment','field_change','feedback','automation'
    )),
    visibility VARCHAR(20) NOT NULL DEFAULT 'internal' CHECK (visibility IN ('public', 'internal')),
    author_type VARCHAR(20) NOT NULL CHECK (author_type IN ('agent', 'contact', 'system', 'automation')),
    author_id UUID,
    body_html TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    email_message_id TEXT,
    email_in_reply_to TEXT,
    email_references TEXT[] NOT NULL DEFAULT '{}',
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'delivered' CHECK (delivery_status IN ('queued', 'sent', 'delivered', 'bounced', 'failed')),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 17. Mentions & Attachments
CREATE TABLE IF NOT EXISTS mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    event_id UUID NOT NULL REFERENCES ticket_events(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    ticket_id UUID REFERENCES tickets(id) ON DELETE CASCADE,
    event_id UUID REFERENCES ticket_events(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    checksum_sha256 VARCHAR(64) NOT NULL,
    is_inline BOOLEAN NOT NULL DEFAULT FALSE,
    content_id TEXT,
    scan_status VARCHAR(20) NOT NULL DEFAULT 'clean' CHECK (scan_status IN ('pending', 'clean', 'infected', 'skipped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 18. Assignment & Automation Rules
CREATE TABLE IF NOT EXISTS assignment_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority_order INT NOT NULL DEFAULT 100,
    trigger VARCHAR(30) NOT NULL CHECK (trigger IN ('on_create', 'on_update', 'on_schedule')),
    match_conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    strategy VARCHAR(30) NOT NULL CHECK (strategy IN ('round_robin', 'least_open', 'least_weighted', 'skill_match', 'specific_agent')),
    target_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    candidate_agent_ids UUID[] NOT NULL DEFAULT '{}',
    active_weekdays INT[] NOT NULL DEFAULT '{0,1,2,3,4,5,6}',
    active_from_time TIME,
    active_to_time TIME,
    respect_availability BOOLEAN NOT NULL DEFAULT TRUE,
    respect_capacity BOOLEAN NOT NULL DEFAULT TRUE,
    last_assigned_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS automation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority_order INT NOT NULL DEFAULT 100,
    trigger VARCHAR(30) NOT NULL CHECK (trigger IN ('ticket_created', 'ticket_updated', 'reply_received', 'status_changed', 'time_since_condition', 'sla_threshold')),
    match_conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions JSONB NOT NULL DEFAULT '{}'::jsonb,
    run_once_per_ticket BOOLEAN NOT NULL DEFAULT FALSE,
    schedule_cron TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS automation_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'skipped')),
    error TEXT,
    actions_applied JSONB NOT NULL DEFAULT '{}'::jsonb,
    ran_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 19. Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    events TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    retry_policy JSONB NOT NULL DEFAULT '{"max_retries": 5, "initial_interval_sec": 5}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event TEXT NOT NULL,
    payload JSONB NOT NULL,
    response_code INT,
    attempt INT NOT NULL DEFAULT 1,
    next_retry_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 20. Knowledge Base
CREATE TABLE IF NOT EXISTS kb_spaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    visibility VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'authenticated', 'internal')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS kb_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    space_id UUID NOT NULL REFERENCES kb_spaces(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES kb_categories(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(space_id, slug)
);

CREATE TABLE IF NOT EXISTS kb_articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    category_id UUID NOT NULL REFERENCES kb_categories(id) ON DELETE CASCADE,
    slug VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'review', 'published', 'archived')),
    locale VARCHAR(10) NOT NULL DEFAULT 'en',
    body_html TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    excerpt TEXT,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ,
    view_count INT NOT NULL DEFAULT 0,
    helpful_count INT NOT NULL DEFAULT 0,
    unhelpful_count INT NOT NULL DEFAULT 0,
    keywords TEXT[] NOT NULL DEFAULT '{}',
    search_vector TSVECTOR,
    translation_of_id UUID REFERENCES kb_articles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(category_id, slug, locale)
);

CREATE TABLE IF NOT EXISTS kb_article_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    article_id UUID NOT NULL REFERENCES kb_articles(id) ON DELETE CASCADE,
    version INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    body_html TEXT NOT NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    change_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE TABLE IF NOT EXISTS kb_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    article_id UUID NOT NULL REFERENCES kb_articles(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    is_helpful BOOLEAN NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 21. Reply Templates (Canned Responses)
CREATE TABLE IF NOT EXISTS reply_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    scope VARCHAR(20) NOT NULL DEFAULT 'global' CHECK (scope IN ('global', 'team', 'personal')),
    owner_agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
    subject_template TEXT NOT NULL DEFAULT '',
    body_template TEXT NOT NULL DEFAULT '',
    attachment_ids UUID[] NOT NULL DEFAULT '{}',
    usage_count INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 22. Mail Accounts, Filters & Suppressions
CREATE TABLE IF NOT EXISTS mail_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    label VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL UNIQUE,
    direction VARCHAR(20) NOT NULL DEFAULT 'both' CHECK (direction IN ('inbound', 'outbound', 'both')),
    is_default_inbound BOOLEAN NOT NULL DEFAULT FALSE,
    is_default_outbound BOOLEAN NOT NULL DEFAULT FALSE,
    protocol VARCHAR(20) NOT NULL CHECK (protocol IN ('imap', 'gmail_oauth', 'ms_oauth', 'smtp', 'ses', 'sendgrid')),
    credentials_encrypted TEXT NOT NULL DEFAULT '',
    poll_interval_seconds INT NOT NULL DEFAULT 60,
    last_polled_at TIMESTAMPTZ,
    last_uid BIGINT NOT NULL DEFAULT 0,
    default_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    default_type_id UUID REFERENCES ticket_types(id) ON DELETE SET NULL,
    strip_quoted_reply BOOLEAN NOT NULL DEFAULT TRUE,
    signature_html TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS mail_filters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    mail_account_id UUID NOT NULL REFERENCES mail_accounts(id) ON DELETE CASCADE,
    match JSONB NOT NULL DEFAULT '{}'::jsonb,
    action JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS mail_suppressions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email VARCHAR(255) NOT NULL UNIQUE,
    reason VARCHAR(20) NOT NULL CHECK (reason IN ('bounce', 'complaint', 'manual')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 23. Field Definitions & Saved Views
CREATE TABLE IF NOT EXISTS field_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    entity VARCHAR(20) NOT NULL CHECK (entity IN ('ticket', 'contact', 'organization', 'article')),
    key VARCHAR(64) NOT NULL,
    label VARCHAR(255) NOT NULL,
    data_type VARCHAR(20) NOT NULL CHECK (data_type IN (
        'text', 'textarea', 'number', 'decimal', 'bool', 'date', 'datetime',
        'select', 'multiselect', 'link', 'email', 'phone', 'url', 'currency', 'json'
    )),
    options JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_agent_only BOOLEAN NOT NULL DEFAULT FALSE,
    show_in_portal_form BOOLEAN NOT NULL DEFAULT TRUE,
    show_in_list BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    validation_regex TEXT,
    depends_on JSONB,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(entity, key)
);

CREATE TABLE IF NOT EXISTS saved_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    entity VARCHAR(20) NOT NULL DEFAULT 'ticket',
    owner_id UUID REFERENCES users(id) ON DELETE CASCADE,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'team', 'public')),
    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    columns JSONB NOT NULL DEFAULT '[]'::jsonb,
    sort JSONB NOT NULL DEFAULT '{}'::jsonb,
    group_by TEXT,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- 24. Notification Preferences & Notifications
CREATE TABLE IF NOT EXISTS notification_prefs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'in_app', 'webpush', 'slack')),
    event_key VARCHAR(64) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    digest VARCHAR(20) NOT NULL DEFAULT 'instant' CHECK (digest IN ('instant', 'hourly', 'daily')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    UNIQUE(user_id, channel, event_key)
);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_key VARCHAR(64) NOT NULL,
    ticket_id UUID REFERENCES tickets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    read_at TIMESTAMPTZ,
    action_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 25. Audit Log (Append-Only)
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    actor_id UUID,
    actor_type VARCHAR(20) NOT NULL,
    entity VARCHAR(64) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(32) NOT NULL,
    before JSONB,
    after JSONB,
    ip INET,
    user_agent TEXT,
    at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 26. Transactional Outbox
CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    aggregate VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    published_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

-- 27. Roles, Permissions & Role Assignments
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    key VARCHAR(100) NOT NULL UNIQUE,
    label VARCHAR(255) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- INDEXES (Spec Requirements)

-- 1. Partial Index for Team Queue
CREATE INDEX IF NOT EXISTS idx_tickets_team_active 
ON tickets (assigned_team_id, updated_at DESC, id DESC) 
WHERE deleted_at IS NULL AND status_category IN ('open', 'pending');

-- 2. Partial Index for Agent Queue
CREATE INDEX IF NOT EXISTS idx_tickets_agent_active 
ON tickets (assigned_agent_id, updated_at DESC, id DESC) 
WHERE deleted_at IS NULL AND status_category IN ('open', 'pending');

-- 3. Partial Index for First Response SLA Due
CREATE INDEX IF NOT EXISTS idx_tickets_first_response_due 
ON tickets (first_response_due_at) 
WHERE first_responded_at IS NULL AND sla_paused_at IS NULL AND deleted_at IS NULL;

-- 4. Partial Index for Resolution SLA Due
CREATE INDEX IF NOT EXISTS idx_tickets_resolution_due 
ON tickets (resolution_due_at) 
WHERE resolved_at IS NULL AND sla_paused_at IS NULL AND deleted_at IS NULL;

-- 5. Full-Text Search GIN Indexes
CREATE INDEX IF NOT EXISTS idx_tickets_search_vector ON tickets USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_kb_articles_search_vector ON kb_articles USING GIN (search_vector);

-- 6. Ticket Events Ordered Timeline
CREATE INDEX IF NOT EXISTS idx_ticket_events_timeline ON ticket_events (ticket_id, occurred_at DESC);

-- 7. BRIN Index on Audit Log
CREATE INDEX IF NOT EXISTS idx_audit_log_at_brin ON audit_log USING BRIN (at);

-- 8. Unique index for inbound email deduplication
CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_events_email_msg_id ON ticket_events (email_message_id) WHERE email_message_id IS NOT NULL;

-- 9. Outbox queue index
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox (created_at ASC) WHERE published_at IS NULL;

-- Full-Text Search Vector Update Functions & Triggers
CREATE OR REPLACE FUNCTION tickets_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.reference_no, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.subject, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.description_text, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_tickets_search_vector
BEFORE INSERT OR UPDATE ON tickets
FOR EACH ROW EXECUTE FUNCTION tickets_search_vector_update();

CREATE OR REPLACE FUNCTION kb_articles_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(array_to_string(NEW.keywords, ' '), '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.body_text, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_kb_articles_search_vector
BEFORE INSERT OR UPDATE ON kb_articles
FOR EACH ROW EXECUTE FUNCTION kb_articles_search_vector_update();
