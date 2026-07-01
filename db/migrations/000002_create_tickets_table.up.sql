CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tickets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number   VARCHAR(20) UNIQUE NOT NULL,
    requester_name  VARCHAR(255) NOT NULL,
    requester_email VARCHAR(255) NOT NULL,
    project_ref     VARCHAR(100),
    category        VARCHAR(20) NOT NULL,
    priority        VARCHAR(20) NOT NULL DEFAULT 'medium',
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    subject         VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL,
    assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
    sla_due_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk query yang paling sering dipakai
CREATE INDEX idx_tickets_ticket_number ON tickets(ticket_number);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);