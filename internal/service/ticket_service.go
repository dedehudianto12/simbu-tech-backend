package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
	"github.com/dedehudianto12/simbu-tech-backend/internal/repository"
)

// NotFoundError is returned when a requested resource does not exist.
type NotFoundError struct {
	Resource string
	Key      string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.Key)
}

// CreateTicketInput holds the fields needed to create a new ticket.
type CreateTicketInput struct {
	RequesterName  string
	RequesterEmail string
	Subject        string
	Description    string
	Category       string
	ProjectRef     *string
}

// TicketService handles all ticket business logic.
type TicketService struct {
	repo   *repository.TicketRepo
	notifSvc *NotificationService
}

// NewTicketService creates a new TicketService.
func NewTicketService(repo *repository.TicketRepo, notif *NotificationService) *TicketService {
	return &TicketService{repo: repo, notifSvc: notif}
}

const ticketNumberChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateTicketNumber() (string, error) {
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(ticketNumberChars))))
		if err != nil {
			return "", fmt.Errorf("generateTicketNumber: %w", err)
		}
		b[i] = ticketNumberChars[n.Int64()]
	}
	return "TCK-" + string(b), nil
}

func calculateSlaDueAt(priority string) time.Time {
	now := time.Now()
	switch priority {
	case model.PriorityLow:
		return now.Add(72 * time.Hour)
	case model.PriorityHigh:
		return now.Add(8 * time.Hour)
	case model.PriorityCritical:
		return now.Add(2 * time.Hour)
	default: // medium
		return now.Add(24 * time.Hour)
	}
}

// CreateTicket creates a new ticket with a random ticket number.
func (s *TicketService) CreateTicket(ctx context.Context, input CreateTicketInput) (model.Ticket, error) {
	priority := model.PriorityMedium
	slaDue := calculateSlaDueAt(priority)
	slaDuePtr := &slaDue

	var ticket model.Ticket
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		ticketNumber, err := generateTicketNumber()
		if err != nil {
			return model.Ticket{}, fmt.Errorf("ticket_service.CreateTicket: %w", err)
		}

		ticket = model.Ticket{
			TicketNumber:   ticketNumber,
			RequesterName:  input.RequesterName,
			RequesterEmail: input.RequesterEmail,
			ProjectRef:     input.ProjectRef,
			Category:       input.Category,
			Priority:       priority,
			Status:         model.StatusOpen,
			Subject:        input.Subject,
			Description:    input.Description,
			SlaDueAt:       slaDuePtr,
		}

		ticket, err = s.repo.Create(ctx, ticket)
		if err == nil {
			break
		}

		// ponytail: string match on dup key; pgx v5 PgError would be cleaner but adds import
		if strings.Contains(err.Error(), "duplicate key") {
			lastErr = err
			continue
		}
		return model.Ticket{}, fmt.Errorf("ticket_service.CreateTicket: %w", err)
	}

	if lastErr != nil && ticket.ID == uuid.Nil {
		return model.Ticket{}, fmt.Errorf("ticket_service.CreateTicket: failed after 3 attempts: %w", lastErr)
	}

	go s.notifSvc.SendTicketCreated(ticket)

	return ticket, nil
}

// GetTicketByNumber returns a ticket by its public ticket number.
func (s *TicketService) GetTicketByNumber(ctx context.Context, ticketNumber string) (model.Ticket, error) {
	ticket, err := s.repo.GetByTicketNumber(ctx, ticketNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, &NotFoundError{Resource: "ticket", Key: ticketNumber}
		}
		return model.Ticket{}, fmt.Errorf("ticket_service.GetTicketByNumber: %w", err)
	}
	return ticket, nil
}

// GetTicketByID returns a ticket by its internal UUID.
func (s *TicketService) GetTicketByID(ctx context.Context, id uuid.UUID) (model.Ticket, error) {
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, &NotFoundError{Resource: "ticket", Key: id.String()}
		}
		return model.Ticket{}, fmt.Errorf("ticket_service.GetTicketByID: %w", err)
	}
	return ticket, nil
}

// ListTickets returns a filtered, paginated list of tickets.
func (s *TicketService) ListTickets(ctx context.Context, filters repository.TicketFilters) ([]model.Ticket, int, error) {
	tickets, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("ticket_service.ListTickets: %w", err)
	}
	return tickets, total, nil
}

var validTransitions = map[string]string{
	model.StatusOpen:       model.StatusInProgress,
	model.StatusInProgress: model.StatusResolved,
	model.StatusResolved:   model.StatusClosed,
}

// UpdateTicketStatus validates and performs a status transition.
func (s *TicketService) UpdateTicketStatus(ctx context.Context, id uuid.UUID, newStatus string, changedBy uuid.UUID) (model.Ticket, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, &NotFoundError{Resource: "ticket", Key: id.String()}
		}
		return model.Ticket{}, fmt.Errorf("ticket_service.UpdateTicketStatus: %w", err)
	}

	allowedNext, ok := validTransitions[current.Status]
	if !ok || allowedNext != newStatus {
		return model.Ticket{}, fmt.Errorf("invalid status transition from %s to %s", current.Status, newStatus)
	}

	if err := s.repo.UpdateStatus(ctx, id, current.Status, newStatus, changedBy); err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_service.UpdateTicketStatus: %w", err)
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_service.UpdateTicketStatus: %w", err)
	}

	go s.notifSvc.SendStatusUpdated(updated)

	return updated, nil
}

// UpdateTicketFields updates mutable fields on a ticket.
// If status is set, delegates to UpdateTicketStatus to enforce transition rules.
func (s *TicketService) UpdateTicketFields(ctx context.Context, id uuid.UUID, updates repository.TicketUpdate, changedBy uuid.UUID) (model.Ticket, error) {
	if updates.Status != nil {
		return s.UpdateTicketStatus(ctx, id, *updates.Status, changedBy)
	}

	ticket, err := s.repo.UpdateFields(ctx, id, updates)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, &NotFoundError{Resource: "ticket", Key: id.String()}
		}
		return model.Ticket{}, fmt.Errorf("ticket_service.UpdateTicketFields: %w", err)
	}
	return ticket, nil
}

// FullTicket holds a ticket with its comments and status history.
type FullTicket struct {
	Ticket   model.Ticket
	Comments []model.TicketComment
	History  []model.TicketStatusHistory
}

// GetFullTicket returns a ticket with its comments and status history.
func (s *TicketService) GetFullTicket(ctx context.Context, id uuid.UUID) (FullTicket, error) {
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FullTicket{}, &NotFoundError{Resource: "ticket", Key: id.String()}
		}
		return FullTicket{}, fmt.Errorf("ticket_service.GetFullTicket: %w", err)
	}

	comments, err := s.repo.GetCommentsByTicketID(ctx, id)
	if err != nil {
		return FullTicket{}, fmt.Errorf("ticket_service.GetFullTicket: %w", err)
	}

	history, err := s.repo.GetStatusHistory(ctx, id)
	if err != nil {
		return FullTicket{}, fmt.Errorf("ticket_service.GetFullTicket: %w", err)
	}

	return FullTicket{Ticket: ticket, Comments: comments, History: history}, nil
}

// AddComment adds a comment to a ticket.
func (s *TicketService) AddComment(ctx context.Context, ticketID uuid.UUID, body, authorType string, authorID *uuid.UUID) (model.TicketComment, error) {
	comment := model.TicketComment{
		TicketID:   ticketID,
		AuthorType: authorType,
		AuthorID:   authorID,
		Body:       body,
	}
	comment, err := s.repo.AddComment(ctx, comment)
	if err != nil {
		// ponytail: check for FK violation to give a nicer error
		if strings.Contains(err.Error(), "violates foreign key") {
			return model.TicketComment{}, &NotFoundError{Resource: "ticket", Key: ticketID.String()}
		}
		return model.TicketComment{}, fmt.Errorf("ticket_service.AddComment: %w", err)
	}
	return comment, nil
}
