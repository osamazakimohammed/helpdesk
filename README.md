# 🎫 Helpdesk - Modern Customer Support Platform

A high-performance, self-hostable, full-stack customer support and ticketing platform built with **Go 1.23**, **PostgreSQL 17**, **pgx/v5**, **sqlc**, and a responsive Single-Page Application (SPA).

---

## ⚡ Tech Stack & Architecture

- **Backend**: Go 1.23 with `chi` router, compiled type-safe SQL queries via `sqlc` + `pgx/v5` connection pooling.
- **Database**: PostgreSQL 17 with UUIDv7 time-ordered primary keys, keyset pagination, and BRIN-indexed audit logs.
- **Caching & PubSub**: Redis 7.
- **Transactional Outbox Engine**: Guaranteed delivery of async events and side-effects using PostgreSQL outbox with `SKIP LOCKED` batch drain workers.
- **Storage**: S3-compatible object storage (MinIO / AWS S3 / Local Disk) with SHA-256 validation.
- **Frontend**: Zero-dependency, responsive dark-mode Single-Page Application (SPA) embedded directly into the Go binary via `embed.FS` or served live from disk in development.
- **Internationalization & RTL**: Full UTF-8 support with native Arabic right-to-left (`dir="auto"`) automatic rendering.
- **API Documentation**: OpenAPI 3.1 specification with integrated Swagger UI at `/api/docs`.

---

## 🌐 Application Surfaces

| Surface | URL Path | Description | Access / Role |
|---|---|---|---|
| **Agent Workspace** | `/app` | Real-time ticket queues, status transitions, conversation stream, internal private notes, customer replies, and team assignment. | `admin`, `agent` |
| **Customer Portal** | `/portal` | Client self-service portal, request tracking, multi-language threaded conversations, and direct replies. | `admin`, `contact` (Customer) |
| **Request Intake Form** | `/submit` | Streamlined intake form with automatic authentication detection and custom field support. | `admin`, `contact`, Public |
| **Knowledge Base** | `/kb` | Fast self-service help center, space and category browsing, full-text search, and article views. | All Roles / Public |
| **Interactive API Docs** | `/api/docs` | OpenAPI 3.1 documentation with interactive Swagger UI. | Public |

---

## 🚀 Quickstart Guide

### 1. Prerequisites
- [Go 1.23+](https://golang.org)
- [Podman](https://podman.io) or [Docker](https://docker.com) & Docker Compose

### 2. Start PostgreSQL & Redis
```bash
podman run -d --name helpdesk-postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=helpdesk postgres:17-alpine
podman run -d --name helpdesk-redis -p 6379:6379 redis:7-alpine
```
*Or using Docker Compose:*
```bash
docker compose up -d
```

### 3. Run Migrations & Seed Database
```bash
# Run database schema migrations
go run cmd/server/main.go migrate

# Seed standard roles, statuses, priorities, and default mock accounts
go run cmd/server/main.go seed
```

### 4. Start the Application Server
```bash
go run cmd/server/main.go serve
```
The server will start at **http://localhost:8080**.

---

## 🔑 Authentication & Testing Credentials

The system supports logging in using **either a plain username or full email address**:

| Role | Username / Email | Password | Default Destination | Accessible Surfaces |
|---|---|---|---|---|
| **Administrator** | `admin` *or* `admin@helpdesk.local` | `admin` | `/app` | **Full Platform Access**: Agent Workspace, Customer Portal, Submit Request, Knowledge Base, API Docs |
| **Support Agent** | `support` *or* `support@helpdesk.local` | `support` | `/app` | Agent Workspace, Help Center |
| **Customer User** | `customer` *or* `customer@helpdesk.local` | `customer` | `/portal` | Customer Portal, Submit Request, Knowledge Base |

---

## 🏛️ Key Architectural Design Rules

1. **Strict Query Budgets**:
   - **Ticket Queues**: Exactly **1 query** with JSON aggregation for tags, assignee, contact, status, and metadata. Keyset pagination (`WHERE (updated_at, id) < ($cursor_time, $cursor_id) ORDER BY updated_at DESC, id DESC LIMIT $limit`) ensures consistent sub-millisecond query latency.
   - **Ticket Detail**: Exactly **2 queries** (Query 1: Ticket + Contact + Org + Team; Query 2: Keyset-paginated timeline events with attachments and mentions).
   - **Query Budget Enforcer**: Middleware attaches `X-DB-Query-Count` to all responses to guarantee query efficiency.

2. **Transactional Outbox Pattern**:
   - All asynchronous operations (email notifications, webhooks, search indexing) are written to the `outbox` table in the same transaction as the ticket state.
   - Dedicated background workers drain the outbox with idempotency and exponential backoff.

3. **UUIDv7 & Audit Integrity**:
   - Time-ordered UUIDv7 primary keys on all tables.
   - Append-only `audit_log` table with `BRIN(at)` index for high-throughput tracking.

---

## 📡 API Overview

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Authenticate user/contact and retrieve JWT token |
| `POST` | `/api/v1/auth/register` | Create a new user account |
| `GET` | `/api/v1/app/tickets` | List agent queue tickets (keyset paginated) |
| `POST` | `/api/v1/app/tickets` | Create a new ticket (agent intake) |
| `GET` | `/api/v1/app/tickets/{id}` | Get ticket detail |
| `PATCH` | `/api/v1/app/tickets/{id}` | Update ticket status, priority, or assignment |
| `GET` | `/api/v1/app/tickets/{id}/events` | List ticket timeline events |
| `POST` | `/api/v1/app/tickets/{id}/events` | Post public reply or team internal note |
| `GET` | `/api/v1/portal/tickets` | List customer portal tickets |
| `POST` | `/api/v1/portal/tickets/{id}/reply` | Submit customer reply on ticket |
| `POST` | `/api/v1/submit/ticket` | Public ticket intake submission |
| `GET` | `/api/v1/kb/spaces` | List knowledge base spaces |
| `GET` | `/api/docs` | Interactive Swagger UI documentation |

---

## 🧪 Testing

Run unit and integration tests:
```bash
go test -v ./...
```

---

## 📄 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
