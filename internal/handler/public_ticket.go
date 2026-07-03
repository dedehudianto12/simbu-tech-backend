package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"github.com/dedehudianto12/simbu-tech-backend/internal/service"
)

type PublicTicketHandler struct {
	svc *service.TicketService
}

func NewPublicTicketHandler(svc *service.TicketService) *PublicTicketHandler {
	return &PublicTicketHandler{svc: svc}
}

var validCategories = []string{"incident", "service_request", "inquiry"}

type createTicketBody struct {
	RequesterName  string  `json:"requester_name"`
	RequesterEmail string  `json:"requester_email"`
	Subject        string  `json:"subject"`
	Description    string  `json:"description"`
	Category       string  `json:"category"`
	ProjectRef     *string `json:"project_ref,omitempty"`
}

func (h *PublicTicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var body createTicketBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.RequesterName == "" {
		writeError(w, http.StatusBadRequest, "field requester_name is required")
		return
	}
	if body.RequesterEmail == "" {
		writeError(w, http.StatusBadRequest, "field requester_email is required")
		return
	}
	if body.Subject == "" {
		writeError(w, http.StatusBadRequest, "field subject is required")
		return
	}
	if body.Description == "" {
		writeError(w, http.StatusBadRequest, "field description is required")
		return
	}
	if body.Category == "" {
		writeError(w, http.StatusBadRequest, "field category is required")
		return
	}
	if !slices.Contains(validCategories, body.Category) {
		writeError(w, http.StatusBadRequest, "field category is invalid")
		return
	}

	input := service.CreateTicketInput{
		RequesterName:  body.RequesterName,
		RequesterEmail: body.RequesterEmail,
		Subject:        body.Subject,
		Description:    body.Description,
		Category:       body.Category,
		ProjectRef:     body.ProjectRef,
	}

	ticket, err := h.svc.CreateTicket(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"ticket_number": ticket.TicketNumber,
		"status":        ticket.Status,
		"created_at":    ticket.CreatedAt,
	})
}

func (h *PublicTicketHandler) GetTicketByNumber(w http.ResponseWriter, r *http.Request) {
	ticketNumber := chi.URLParam(r, "ticketNumber")

	ticket, err := h.svc.GetTicketByNumber(r.Context(), ticketNumber)
	if err != nil {
		var notFoundErr *service.NotFoundError
		if errors.As(err, &notFoundErr) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket_number": ticket.TicketNumber,
		"subject":       ticket.Subject,
		"status":        ticket.Status,
		"category":      ticket.Category,
		"priority":      ticket.Priority,
		"created_at":    ticket.CreatedAt,
		"updated_at":    ticket.UpdatedAt,
	})
}
