# Simbu Ticketing API — Project Context for AI Agents

This file provides full context for AI coding agents (Claude Code, etc.) working on this project.
Read this ENTIRELY before writing any code or making any changes.

---

## 1. What This Project Is

A **helpdesk/ticketing backend** for PT Simbu Teknologi Indonesia — an IT solutions company
(infrastructure, cybersecurity, area security, software development).

This is intentionally a **v1 MVP**. Do not over-engineer. Do not add features not listed here.

**Business goal:** Allow external clients to submit support tickets and track their status
using a ticket number, while internal staff manage tickets via a protected admin dashboard.

**Technical goal (secondary):** This project is also a portfolio piece for the developer,
demonstrating backend Go proficiency for job applications.

---

## 2. Architecture Decisions (FINAL — do not change without explicit instruction)

| Decision             | Choice                                | Reason                                                |
| -------------------- | ------------------------------------- | ----------------------------------------------------- |
| Architecture         | Monolith                              | Simple scope, small team, no reason for microservices |
| Language             | Go                                    | Portfolio goal + performance                          |
| HTTP Framework       | chi                                   | Idiomatic Go, closer to net/http stdlib               |
| Database             | PostgreSQL                            | Relational data, Railway-hosted                       |
| DB Access            | pgx/v5 + sqlc                         | Type-safe, no ORM magic                               |
| Auth                 | JWT (admin only)                      | Public endpoints have NO auth                         |
| Public ticket access | Ticket number only                    | No login required for status check                    |
| Ticket number format | Random (e.g. TCK-7F3K9X2A)            | Sequential IDs are guessable — security risk          |
| Notifications        | Email only (v1)                       | WhatsApp deferred to v2                               |
| Deployment           | Railway (backend) + Vercel (frontend) | Existing stack                                        |

---

## 3. Folder Structure

```
simbu-ticketing-api/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint: load env, connect DB, setup router, start server
├── internal/
│   ├── handler/
│   │   ├── public_ticket.go     # POST /api/public/tickets, GET /api/public/tickets/:ticket_number
│   │   ├── admin_ticket.go      # GET/PATCH /api/admin/tickets (JWT protected)
│   │   └── auth.go              # POST /api/admin/auth/login, /refresh
│   ├── service/
│   │   ├── ticket_service.go    # Business logic: generate ticket_number, calculate sla_due_at
│   │   └── notification_service.go  # Send email via SMTP (async goroutine)
│   ├── repository/
│   │   ├── ticket_repo.go       # DB queries for tickets, comments, attachments, history
│   │   └── user_repo.go         # DB queries for users (staff only)
│   ├── middleware/
│   │   ├── auth.go              # JWT verification for /api/admin/* routes
│   │   └── cors.go              # CORS for Nuxt frontend origins
│   └── model/
│       ├── ticket.go            # Ticket, TicketComment, TicketAttachment, TicketStatusHistory structs
│       └── user.go              # User struct with Role enum
├── pkg/
│   ├── jwt/
│   │   └── jwt.go               # Generate/verify JWT, reusable, no internal/ dependencies
│   └── validator/
│       └── validator.go         # Input validation helpers
├── migrations/
│   └── 000001_create_tickets_table.up.sql   # Run with golang-migrate
│   └── 000001_create_tickets_table.down.sql
├── .env.example
├── .gitignore
├── CLAUDE.md                    # This file
└── README.md
```

**Layer rules (STRICT):**

- `handler` calls `service`, NEVER `repository` directly
- `service` calls `repository`, contains all business logic
- `repository` only does DB queries, NO business logic
- `pkg/` must NOT import anything from `internal/`

---

## 4. Database Schema

### tickets

```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
ticket_number   VARCHAR(20) UNIQUE NOT NULL   -- e.g. TCK-7F3K9X2A (random, NOT sequential)
requester_name  VARCHAR(255) NOT NULL
requester_email VARCHAR(255) NOT NULL         -- used for email notifications
project_ref     VARCHAR(100)                  -- optional, manual entry, no validation
category        VARCHAR(20) NOT NULL          -- enum: incident | service_request | inquiry
priority        VARCHAR(20) NOT NULL DEFAULT 'medium'  -- enum: low | medium | high | critical
status          VARCHAR(20) NOT NULL DEFAULT 'open'    -- enum: open | in_progress | resolved | closed
subject         VARCHAR(255) NOT NULL
description     TEXT NOT NULL
assigned_to     UUID REFERENCES users(id)     -- nullable, set by admin
sla_due_at      TIMESTAMPTZ                   -- calculated from priority on create
created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
```

### users (staff only — NO customer accounts)

```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
name            VARCHAR(255) NOT NULL
email           VARCHAR(255) UNIQUE NOT NULL
password_hash   VARCHAR(255) NOT NULL         -- bcrypt
role            VARCHAR(20) NOT NULL          -- enum: admin | technician | supervisor
created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
```

### ticket_comments

```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE
author_type     VARCHAR(20) NOT NULL          -- enum: customer | staff
author_id       UUID                          -- nullable for customer (no account)
body            TEXT NOT NULL
created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
```

### ticket_attachments

```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE
comment_id      UUID REFERENCES ticket_comments(id)  -- nullable
file_url        TEXT NOT NULL                 -- Cloudflare R2 URL
uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
```

### ticket_status_history

```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE
old_status      VARCHAR(20) NOT NULL
new_status      VARCHAR(20) NOT NULL
changed_by      UUID NOT NULL REFERENCES users(id)
changed_at      TIMESTAMPTZ NOT NULL DEFAULT now()
```

### SLA rules (used in ticket_service.go)

| Priority | SLA Duration |
| -------- | ------------ |
| low      | 72 hours     |
| medium   | 24 hours     |
| high     | 8 hours      |
| critical | 2 hours      |

---

## 5. API Endpoints

### Public (NO auth required)

```
POST   /api/public/tickets
       Body: { requester_name, requester_email, subject, description, category, project_ref? }
       Response: { ticket_number, status, created_at }

GET    /api/public/tickets/:ticket_number
       Response: { ticket_number, subject, status, category, priority, created_at, updated_at }
```

### Admin (JWT Bearer token required)

```
POST   /api/admin/auth/login
       Body: { email, password }
       Response: { access_token, refresh_token }

POST   /api/admin/auth/refresh
       Body: { refresh_token }
       Response: { access_token }

GET    /api/admin/tickets
       Query: ?status=open&priority=high&assigned_to=<uuid>&page=1&limit=20
       Response: paginated list of tickets

GET    /api/admin/tickets/:id
       Response: full ticket detail with comments and status history

PATCH  /api/admin/tickets/:id
       Body: { status?, priority?, assigned_to? }
       Response: updated ticket

POST   /api/admin/tickets/:id/comments
       Body: { body }
       Response: created comment

GET    /api/admin/dashboard/stats
       Response: { open, in_progress, resolved, closed, avg_resolution_hours }
```

---

## 6. Environment Variables

See `.env.example` for all required vars. Key ones:

```
PORT=8080
DATABASE_URL=postgres://simbu:simbu123@localhost:5433/simbu_ticketing?sslmode=disable
JWT_SECRET=<random string, min 32 chars>
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=7d
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM_EMAIL=support@simbu.co.id
ALLOWED_ORIGINS=http://localhost:3000
```

---

## 7. Key Implementation Notes

### Ticket number generation (ticket_service.go)

- Format: `TCK-` + 8 random alphanumeric uppercase chars (e.g. `TCK-7F3K9X2A`)
- Generated in Go, NOT by the database
- Must check uniqueness before insert (retry if collision, extremely rare)
- Use `crypto/rand`, NOT `math/rand` (cryptographically random)

### Notification flow (notification_service.go)

- Called as a goroutine (non-blocking) after ticket create or status update
- v1: email only via SMTP
- Triggers: (1) ticket created → email to requester, (2) status changed → email to requester
- Do NOT block the API response waiting for email to send

### JWT strategy

- Access token: short TTL (15m), used for every admin API call
- Refresh token: long TTL (7d), used only to get new access token
- Store refresh token in httpOnly cookie (more secure than localStorage)

### context.Context usage

- Every DB query MUST receive ctx as first parameter
- ctx comes from r.Context() in handlers, passed down to service, then to repository
- This enables automatic query cancellation if client disconnects

### Error handling pattern

- Repository returns (result, error) — never panics
- Service wraps errors with context: fmt.Errorf("ticket_service.Create: %w", err)
- Handler translates errors to HTTP status codes
- Do NOT expose raw DB errors to the client

---

## 8. What Has Been Done

- [x] Folder structure created
- [x] go.mod initialized
- [x] Dependencies installed (chi, pgx/v5, jwt, bcrypt, godotenv, uuid)
- [x] Migration file 000001 created (tickets table — basic version)
- [x] PostgreSQL running locally on port 5433
- [x] Database: simbu_ticketing, user: simbu

## 9. What Needs To Be Done (in order)

- [ ] Complete migration files for ALL tables (users, ticket_comments, ticket_attachments, ticket_status_history)
- [ ] Implement main.go (load env, connect DB pool, setup router, start server)
- [ ] Implement model structs (ticket.go, user.go)
- [ ] Implement repository layer (ticket_repo.go, user_repo.go)
- [ ] Implement service layer (ticket_service.go, notification_service.go)
- [ ] Implement middleware (auth.go JWT verify, cors.go)
- [ ] Implement handlers (public_ticket.go, admin_ticket.go, auth.go)
- [ ] Test all endpoints manually (curl or Postman)
- [ ] Write .env for production (Railway deployment)
- [ ] Deploy to Railway

---

## 10. What NOT To Do

- Do NOT use GORM or any ORM — use pgx/v5 directly or sqlc
- Do NOT put business logic in repository layer
- Do NOT put DB queries in handler layer
- Do NOT return raw PostgreSQL errors to API clients
- Do NOT use math/rand for ticket number generation (use crypto/rand)
- Do NOT add features outside this spec without asking the developer first
- Do NOT use sequential ticket numbers (security risk — guessable)
- Do NOT add customer login/auth — public access is ticket number only (v1 decision)
