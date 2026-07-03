package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dedehudianto12/simbu-tech-backend/internal/middleware"
	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
	"github.com/dedehudianto12/simbu-tech-backend/internal/repository"
	"github.com/dedehudianto12/simbu-tech-backend/internal/service"
)

type AdminTicketHandler struct {
	svc *service.TicketService
}

func NewAdminTicketHandler(svc *service.TicketService) *AdminTicketHandler {
	return &AdminTicketHandler{svc: svc}
}

func (h *AdminTicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}

	filters := repository.TicketFilters{
		Page:  page,
		Limit: limit,
	}

	if s := q.Get("status"); s != "" {
		filters.Status = &s
	}
	if p := q.Get("priority"); p != "" {
		filters.Priority = &p
	}
	if a := q.Get("assigned_to"); a != "" {
		id, err := uuid.Parse(a)
		if err == nil {
			filters.AssignedTo = &id
		}
	}

	tickets, total, err := h.svc.ListTickets(r.Context(), filters)
	if err != nil {
		log.Printf("ListTickets: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  tickets,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminTicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	full, err := h.svc.GetFullTicket(r.Context(), id)
	if err != nil {
		var notFoundErr *service.NotFoundError
		if errors.As(err, &notFoundErr) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		log.Printf("GetTicket: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":   full.Ticket,
		"comments": full.Comments,
		"history":  full.History,
	})
}

type updateTicketBody struct {
	Status     *string `json:"status,omitempty"`
	Priority   *string `json:"priority,omitempty"`
	AssignedTo *string `json:"assigned_to,omitempty"`
}

func (h *AdminTicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var body updateTicketBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	changedBy, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updates := repository.TicketUpdate{
		Status:   body.Status,
		Priority: body.Priority,
	}
	if body.AssignedTo != nil {
		uid, err := uuid.Parse(*body.AssignedTo)
		if err == nil {
			updates.AssignedTo = &uid
		}
	}

	ticket, err := h.svc.UpdateTicketFields(r.Context(), id, updates, changedBy)
	if err != nil {
		var notFoundErr *service.NotFoundError
		if errors.As(err, &notFoundErr) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		if strings.Contains(err.Error(), "invalid status transition") {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		log.Printf("UpdateTicket: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

type addCommentBody struct {
	Body string `json:"body"`
}

func (h *AdminTicketHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var body addCommentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Body == "" {
		writeError(w, http.StatusBadRequest, "field body is required")
		return
	}

	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	comment, err := h.svc.AddComment(r.Context(), id, body.Body, model.AuthorTypeStaff, &userUUID)
	if err != nil {
		var notFoundErr *service.NotFoundError
		if errors.As(err, &notFoundErr) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		log.Printf("AddComment: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

func (h *AdminTicketHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	statuses := []string{model.StatusOpen, model.StatusInProgress, model.StatusResolved, model.StatusClosed}
	results := make(map[string]int, 4)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, s := range statuses {
		wg.Add(1)
		go func(status string) {
			defer wg.Done()
			_, total, err := h.svc.ListTickets(r.Context(), repository.TicketFilters{
				Status: &status,
				Page:   1,
				Limit:  1,
			})
			if err != nil {
				log.Printf("GetStats %s: %v", status, err)
				return
			}
			mu.Lock()
			results[status] = total
			mu.Unlock()
		}(s)
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]any{
		"open":        results[model.StatusOpen],
		"in_progress": results[model.StatusInProgress],
		"resolved":    results[model.StatusResolved],
		"closed":      results[model.StatusClosed],
	})
}
