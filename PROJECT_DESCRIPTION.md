# 🎫 Helpdesk Platform — Comprehensive Project Description & Technical Specification

---

## 📌 1. Executive Overview

The **Digitera Helpdesk Platform** is a high-performance, self-hostable, full-stack customer support and ticket management system. Built with an emphasis on **strict query budgeting**, **transactional safety**, **sub-millisecond latency**, and **operational simplicity**, the platform eliminates the bloated dependencies, complex build steps, and slow rendering loops typical of legacy customer support tools.

The application ships as a **single, self-contained binary** where the compiled Go backend embeds the entire reactive Single-Page Application (SPA), providing an out-of-the-box support suite with zero client-side framework overhead.

---

## ⚡ 2. Complete Technology Stack

```
                                  ┌─────────────────────────────────────────┐
                                  │      Embedded Vanilla SPA Frontend      │
                                  │  (HTML5, Vanilla CSS, JS, RTL dir=auto) │
                                  └────────────────────┬────────────────────┘
                                                       │ HTTP / JSON (REST)
                                                       ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                              Go 1.23 HTTP Backend                                                │
│  ┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────────┐  │
│  │   Chi v5 Router       │  │  Query Budget Guard   │  │   JWT Auth Engine     │  │   OpenAPI 3.1 Engine      │  │
│  │ (Zero-Alloc Routing)  │  │ (X-DB-Query-Count)    │  │ (Dual Username/Email) │  │  (Interactive Swagger UI) │  │
│  └───────────────────────┘  └───────────────────────┘  └───────────────────────┘  └───────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │                         SQLC Type-Safe Compiled Queries + pgx/v5 Connection Pool                            │  │
│  └───────────────────────────────────────────────────────┬─────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┘
                                                           │
                                ┌──────────────────────────┴──────────────────────────┐
                                ▼                                                     ▼
┌────────────────────────────────────────────────────────────────┐  ┌──────────────────────────────────────────────┐
│                    PostgreSQL 17 Database                      │  │                   Redis 7                    │
│ • UUIDv7 Time-Ordered Primary Keys                             │  │ • Rate Limiting & Token Bucket               │
│ • Keyset-Paginated Queries (WHERE (updated_at, id) < ($t, $id)│  │ • Distributed Lock Coordination              │
│ • Transactional Outbox Table (FOR UPDATE SKIP LOCKED)          │  │ • Real-Time PubSub & Queue Draining          │
│ • BRIN Range Index on Immutable Audit Log                      │  └──────────────────────────────────────────────┘
└────────────────────────────────────────────────────────────────┘
```

### 🔹 Backend Core
- **Language & Runtime**: **Go 1.23** (Compiled, statically typed, zero runtime overhead).
- **HTTP Routing**: `github.com/go-chi/chi/v5` — fast, lightweight, idiomatic HTTP router.
- **Database Driver & Pooling**: `github.com/jackc/pgx/v5` + `puddle` connection pool for high-concurrency database throughput.
- **Query Compilation**: **SQLC** — compiles raw SQL queries into compile-time type-safe Go code with zero ORM abstraction overhead.
- **Authentication**: `github.com/golang-jwt/jwt/v5` — cryptographically signed JWT tokens with claims and role management.
- **HTML Sanitization**: `github.com/microcosm-cc/bluemonday` — robust XSS protection for rich text descriptions and comments.
- **CORS & Middleware**: `github.com/go-chi/cors` with configurable origin, header exposure, and query budget interceptors.

### 🔹 Database & Storage Layer
- **Relational Database**: **PostgreSQL 17** (Alpine).
- **Primary Keys**: **UUIDv7** — 128-bit time-ordered UUIDs providing natural chronological B-Tree insertion without sequential ID enumeration risks.
- **Keyset Pagination**: Strictly cursor-based pagination `(updated_at, id) < (cursor_time, cursor_id)` ensuring sub-millisecond retrieval on million-row tables without `OFFSET` degradation.
- **Audit Trails**: Append-only `audit_log` table with **BRIN (Block Range Index)** on timestamp for compact, high-speed chronological scans.
- **Object Storage Support**: S3-compatible interface (AWS S3 / MinIO / Local Disk) with SHA-256 integrity validation.

### 🔹 Concurrency & Message Bus
- **Cache & PubSub**: **Redis 7** (Alpine) for token-bucket rate limiting and async job coordination.
- **Transactional Outbox Engine**: Outbox pattern with `FOR UPDATE SKIP LOCKED` batch drain workers to guarantee at-least-once delivery for asynchronous side effects.

### 🔹 Frontend Architecture
- **Framework**: **Vanilla Single-Page Application (SPA)** — zero external npm/bundler dependencies.
- **Rendering**: Reactive DOM updates with client-side hash/path router.
- **Styling**: Vanilla CSS Design Token System with Tailwind-inspired dark mode surfaces, sleek gradients, and glassmorphism.
- **Distribution**: Packaged directly into the Go binary using `//go:embed all:dist` via Go's `embed.FS`.
- **Live Development**: Live disk file watcher with automatic cache-busting and no-cache dev headers.

---

## 🚀 3. Comprehensive Feature Breakdown

### 🎯 1. Agent Workspace (`/app`)
Designed for support engineers and agents to triage, respond, and resolve customer requests with maximum velocity.
- **Unified Queues**:
  - `📥 All Open`: Instant view of active `new` and `open` tickets.
  - `⏳ Pending Response`: Tracks tickets awaiting customer feedback (`pending` and `on_hold`).
  - `✅ Resolved / Closed`: Displays all completed and finalized tickets (`resolved` and `closed`).
  - `📁 All Tickets`: Master queue showing all records across all statuses.
- **Real-Time Ticket Details**:
  - Live metadata inspector showing Contact Name, Email, Priority, Status, Assignee, and Creation Date.
  - Instant status transition dropdown mapped directly to database timestamps (`resolved_at`, `closed_at`).
- **Interactive Dual-Mode Composer**:
  - **Public Reply**: Direct message sent to the customer with timeline logging.
  - **Internal Note**: Private, team-only note with dedicated amber badge for internal collaboration.
  - **Fast Triage**: Supports <kbd>Ctrl + Enter</kbd> instant submission and in-flight loading indicators (`"Sending..."`).
- **Chronological Timeline Stream**:
  - Keyset-paginated message history rendering customer inquiries, public replies, internal notes, and system audit events.

---

### 🌐 2. Customer Portal (`/portal`)
A clean self-service hub where customers can track all their open and past requests.
- **Ticket History**: Overview card list displaying reference numbers (`TCK-10000`), subjects, statuses, and last activity timestamps.
- **Interactive Discussion Thread**: Customers can view replies from support staff and reply directly within the existing ticket thread.
- **Intake Fallback & Guest Resolution**: Automatically associates tickets created via guest or authenticated intake with the customer's contact record.

---

### 📝 3. Request Intake Form (`/submit`)
A streamlined public and authenticated intake form for submitting support tickets.
- **Automatic Identity Binding**: Pre-fills authenticated user credentials or dynamically provisions guest contact records in PostgreSQL.
- **Priority Selector**: Supports urgency classifications (`Low`, `Medium`, `High`, `Urgent`).
- **Rich Input & Instant Routing**: Sanitizes description input and immediately redirects the user to their ticket view upon submission.

---

### 📚 4. Knowledge Base & Help Center (`/kb`)
A fast public-facing knowledge base for self-service documentation.
- **Spaces & Categories**: Organizes documentation into searchable spaces and topical collections.
- **Full-Text Search**: Live article search bar with indexed keywords.
- **Universal Accessibility**: Accessible to guests, customers, agents, and administrators.

---

### ⚡ 5. Interactive API Documentation (`/api/docs`)
- **OpenAPI 3.1 Specification**: Complete JSON schema specification served at `/openapi.json`.
- **Integrated Swagger UI**: Interactive, browser-based API testing client embedded directly at `/api/docs`.

---

### 🌍 6. Internationalization & Native Arabic (RTL) Support
- **Full UTF-8 Persistence**: Unrestricted support for Arabic, Hebrew, and global character sets across PostgreSQL, Go handlers, and JSON payloads.
- **Bidirectional Text Rendering (`dir="auto"`)**: All ticket subjects, descriptions, timeline items, and input textareas dynamically adapt their alignment to Right-to-Left (RTL) when Arabic content is detected.

---

### 🔑 7. Universal Authentication & Roles
- **Dual-Mode Login**: Users can sign in using **either their plain username** (`admin`, `support`, `customer`) **or their full email address** (`admin@helpdesk.local`, etc.).
- **Role-Based Access Control (RBAC)**:
  - **`admin` / `manager`**: **Universal Platform Access** — unrestricted access to Agent Workspace, Customer Portal, Submit Request, Knowledge Base, and API Docs.
  - **`agent`**: Agent Workspace and Help Center access.
  - **`customer` / `contact`**: Customer Portal, Request Submission, and Knowledge Base access.

---

## 🏛️ 4. Performance Engineering & Architectural Guarantees

| Metric / Rule | Implementation & Guarantee |
|---|---|
| **Ticket List Query Budget** | **Strictly 1 Query**: Keyset pagination with JSON/CTE subquery aggregation for tags, assignees, contacts, and statuses in a single round-trip. |
| **Ticket Detail Query Budget** | **Strictly 2 Queries**: Query 1 fetches ticket metadata & associations; Query 2 retrieves keyset-paginated timeline events. |
| **Query Budget Enforcement** | Custom HTTP middleware verifies query count per request and emits `X-DB-Query-Count` in response headers. |
| **Pagination Scalability** | Cursor-based `(updated_at, id)` keyset pagination eliminates $O(N)$ scan penalties of `OFFSET`. |
| **Outbox Atomic Safety** | Ticket mutations and asynchronous outbox events are committed inside the exact same database transaction. |
| **Audit Log Compression** | High-throughput `audit_log` uses **BRIN (Block Range Index)** to compress millions of log entries into minimal disk space. |

---

## 🔑 5. Pre-Seeded Testing Accounts

| Role | Username / Email | Password | Default Landing | Accessible Views |
|---|---|---|---|---|
| **Administrator** | `admin` *or* `admin@helpdesk.local` | `admin` | `/app` | `/app`, `/portal`, `/submit`, `/kb`, `/api/docs` |
| **Support Agent** | `support` *or* `support@helpdesk.local` | `support` | `/app` | `/app`, `/kb` |
| **Customer User** | `customer` *or* `customer@helpdesk.local` | `customer` | `/portal` | `/portal`, `/submit`, `/kb` |

---

## 📦 6. Deployment & Source Repository

- **GitHub Repository**: [https://github.com/osamazakimohammed/helpdesk](https://github.com/osamazakimohammed/helpdesk)
- **Local Application Server**: `http://localhost:8080`
- **PDF Project Presentation**: [`Digitera_Helpdesk_Project_Presentation.pdf`](file:///run/media/osamazaki/OSAMA/Digitera/helpdesk/Digitera_Helpdesk_Project_Presentation.pdf)
