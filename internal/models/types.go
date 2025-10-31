package models


import (
	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	ExpiresInSeconds  *int   `json:"expires_in_seconds,omitempty"` // optional
}

type LoginResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Token 	  string    `json:"token"`
}

type ChirpRequest struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}