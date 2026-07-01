package model

import (
	"time"

	"github.com/google/uuid"
)

// Role enum
const (
	RoleAdmin      = "admin"
	RoleTechnician = "technician"
	RoleSupervisor = "supervisor"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}
