package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/dedehudianto12/simbu-tech-backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "field email is required")
		return
	}
	if body.Password == "" {
		writeError(w, http.StatusBadRequest, "field password is required")
		return
	}

	accessToken, refreshToken, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		log.Printf("Login: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body refreshBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "field refresh_token is required")
		return
	}

	accessToken, err := h.svc.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
	})
}
