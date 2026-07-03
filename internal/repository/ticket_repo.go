package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
)

type TicketFilters struct {
	Status     *string
	Priority   *string
	AssignedTo *uuid.UUID
	Page       int
	Limit      int
}

type TicketUpdate struct {
	Status     *string
	Priority   *string
	AssignedTo *uuid.UUID
}

type TicketRepo struct {
	pool *pgxpool.Pool
}

func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{pool: pool}
}

func (r *TicketRepo) Create(ctx context.Context, t model.Ticket) (model.Ticket, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tickets (ticket_number, requester_name, requester_email, project_ref, category, priority, status, subject, description, assigned_to, sla_due_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`,
		t.TicketNumber, t.RequesterName, t.RequesterEmail, t.ProjectRef, t.Category, t.Priority, t.Status, t.Subject, t.Description, t.AssignedTo, t.SlaDueAt,
	)
	if err := row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_repo.Create: %w", err)
	}
	return t, nil
}

func (r *TicketRepo) GetByTicketNumber(ctx context.Context, ticketNumber string) (model.Ticket, error) {
	var t model.Ticket
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticket_number, requester_name, requester_email, project_ref, category, priority, status, subject, description, assigned_to, sla_due_at, created_at, updated_at
		FROM tickets WHERE ticket_number = $1`, ticketNumber).
		Scan(&t.ID, &t.TicketNumber, &t.RequesterName, &t.RequesterEmail, &t.ProjectRef, &t.Category, &t.Priority, &t.Status, &t.Subject, &t.Description, &t.AssignedTo, &t.SlaDueAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_repo.GetByTicketNumber: %w", err)
	}
	return t, nil
}

func (r *TicketRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Ticket, error) {
	var t model.Ticket
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticket_number, requester_name, requester_email, project_ref, category, priority, status, subject, description, assigned_to, sla_due_at, created_at, updated_at
		FROM tickets WHERE id = $1`, id).
		Scan(&t.ID, &t.TicketNumber, &t.RequesterName, &t.RequesterEmail, &t.ProjectRef, &t.Category, &t.Priority, &t.Status, &t.Subject, &t.Description, &t.AssignedTo, &t.SlaDueAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_repo.GetByID: %w", err)
	}
	return t, nil
}

func (r *TicketRepo) List(ctx context.Context, f TicketFilters) ([]model.Ticket, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}

	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tickets
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::text IS NULL OR priority = $2)
		  AND ($3::uuid IS NULL OR assigned_to = $3)`,
		f.Status, f.Priority, f.AssignedTo,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("ticket_repo.List: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticket_number, requester_name, requester_email, project_ref, category, priority, status, subject, description, assigned_to, sla_due_at, created_at, updated_at
		FROM tickets
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::text IS NULL OR priority = $2)
		  AND ($3::uuid IS NULL OR assigned_to = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		f.Status, f.Priority, f.AssignedTo, f.Limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ticket_repo.List: %w", err)
	}
	defer rows.Close()

	tickets, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Ticket])
	if err != nil {
		return nil, 0, fmt.Errorf("ticket_repo.List: %w", err)
	}
	if tickets == nil {
		tickets = []model.Ticket{}
	}
	return tickets, total, nil
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, changedBy uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ticket_repo.UpdateStatus: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3 AND status = $4`, newStatus, time.Now(), id, oldStatus)
	if err != nil {
		return fmt.Errorf("ticket_repo.UpdateStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ticket_repo.UpdateStatus: ticket not found or status already changed")
	}

	_, err = tx.Exec(ctx, `INSERT INTO ticket_status_history (ticket_id, old_status, new_status, changed_by) VALUES ($1, $2, $3, $4)`, id, oldStatus, newStatus, changedBy)
	if err != nil {
		return fmt.Errorf("ticket_repo.UpdateStatus: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ticket_repo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *TicketRepo) UpdateFields(ctx context.Context, id uuid.UUID, u TicketUpdate) (model.Ticket, error) {
	var t model.Ticket
	err := r.pool.QueryRow(ctx, `
		UPDATE tickets SET
			status    = COALESCE($1, status),
			priority  = COALESCE($2, priority),
			assigned_to = COALESCE($3, assigned_to),
			updated_at = $4
		WHERE id = $5
		RETURNING id, ticket_number, requester_name, requester_email, project_ref, category, priority, status, subject, description, assigned_to, sla_due_at, created_at, updated_at`,
		u.Status, u.Priority, u.AssignedTo, time.Now(), id,
	).Scan(&t.ID, &t.TicketNumber, &t.RequesterName, &t.RequesterEmail, &t.ProjectRef, &t.Category, &t.Priority, &t.Status, &t.Subject, &t.Description, &t.AssignedTo, &t.SlaDueAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("ticket_repo.UpdateFields: %w", err)
	}
	return t, nil
}

func (r *TicketRepo) AddComment(ctx context.Context, c model.TicketComment) (model.TicketComment, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ticket_comments (ticket_id, author_type, author_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		c.TicketID, c.AuthorType, c.AuthorID, c.Body,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return model.TicketComment{}, fmt.Errorf("ticket_repo.AddComment: %w", err)
	}
	return c, nil
}

func (r *TicketRepo) GetCommentsByTicketID(ctx context.Context, ticketID uuid.UUID) ([]model.TicketComment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticket_id, author_type, author_id, body, created_at
		FROM ticket_comments WHERE ticket_id = $1
		ORDER BY created_at ASC`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket_repo.GetCommentsByTicketID: %w", err)
	}
	defer rows.Close()

	comments, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.TicketComment])
	if err != nil {
		return nil, fmt.Errorf("ticket_repo.GetCommentsByTicketID: %w", err)
	}
	if comments == nil {
		comments = []model.TicketComment{}
	}
	return comments, nil
}

func (r *TicketRepo) GetStatusHistory(ctx context.Context, ticketID uuid.UUID) ([]model.TicketStatusHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticket_id, old_status, new_status, changed_by, changed_at
		FROM ticket_status_history WHERE ticket_id = $1
		ORDER BY changed_at ASC`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket_repo.GetStatusHistory: %w", err)
	}
	defer rows.Close()

	history, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.TicketStatusHistory])
	if err != nil {
		return nil, fmt.Errorf("ticket_repo.GetStatusHistory: %w", err)
	}
	if history == nil {
		history = []model.TicketStatusHistory{}
	}
	return history, nil
}
