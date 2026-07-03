package repository

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
)

func TestRepositories(t *testing.T) {
	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	ticketRepo := NewTicketRepo(pool)
	userRepo := NewUserRepo(pool)

	ticket := model.Ticket{
		TicketNumber:   "TCK-SMOKE1",
		RequesterName:  "Smoke Test",
		RequesterEmail: "smoke@test.com",
		Category:       model.CategoryIncident,
		Priority:       model.PriorityMedium,
		Status:         model.StatusOpen,
		Subject:        "Smoke test",
		Description:    "Testing repo layer",
	}

	created, err := ticketRepo.Create(ctx, ticket)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	log.Printf("✓ Created: %s", created.ID)

	fetched, err := ticketRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.Subject != "Smoke test" {
		t.Errorf("wrong subject: %s", fetched.Subject)
	}

	byNum, err := ticketRepo.GetByTicketNumber(ctx, "TCK-SMOKE1")
	if err != nil {
		t.Fatalf("GetByTicketNumber: %v", err)
	}
	if byNum.ID != created.ID {
		t.Error("GetByTicketNumber returned wrong ticket")
	}

	tickets, total, err := ticketRepo.List(ctx, TicketFilters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 1 {
		t.Error("List returned no tickets")
	}
	_ = tickets

	updated, err := ticketRepo.UpdateFields(ctx, created.ID, TicketUpdate{
		Status: ptr(model.StatusInProgress),
	})
	if err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	if updated.Status != model.StatusInProgress {
		t.Errorf("UpdateFields: expected in_progress, got %s", updated.Status)
	}

	comment, err := ticketRepo.AddComment(ctx, model.TicketComment{
		TicketID:   created.ID,
		AuthorType: model.AuthorTypeStaff,
		Body:       "Test comment",
	})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if comment.ID == uuid.Nil {
		t.Error("AddComment returned zero UUID")
	}

	comments, err := ticketRepo.GetCommentsByTicketID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCommentsByTicketID: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}

	history, err := ticketRepo.GetStatusHistory(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetStatusHistory: %v", err)
	}
	_ = history

	_, err = userRepo.GetByEmail(ctx, "nobody@test.com")
	if err == nil {
		t.Error("GetByEmail should return error for nonexistent user")
	}

	log.Println("All smoke tests passed!")
}

func ptr[T any](v T) *T { return &v }

