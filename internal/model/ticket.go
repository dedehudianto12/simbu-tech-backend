package model

import (
	"time"

	"github.com/google/uuid"
)

// Status enum
const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

// Category enum
const (
	CategoryIncident       = "incident"
	CategoryServiceRequest = "service_request"
	CategoryInquiry        = "inquiry"
)

// Priority enum
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

type Ticket struct {
	ID             uuid.UUID
	TicketNumber   string
	RequesterName  string
	RequesterEmail string
	ProjectRef     *string
	Category       string
	Priority       string
	Status         string
	Subject        string
	Description    string
	AssignedTo     *uuid.UUID
	SlaDueAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AuthorType enum
const (
	AuthorTypeCustomer = "customer"
	AuthorTypeStaff    = "staff"
)

type TicketComment struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	AuthorType string
	AuthorID   *uuid.UUID
	Body       string
	CreatedAt  time.Time
}

type TicketAttachment struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	CommentID  *uuid.UUID
	FileURL    string
	UploadedAt time.Time
}

type TicketStatusHistory struct {
	ID        uuid.UUID
	TicketID  uuid.UUID
	OldStatus string
	NewStatus string
	ChangedBy uuid.UUID
	ChangedAt time.Time
}
