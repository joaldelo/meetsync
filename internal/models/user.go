package models

import (
	"time"
)

// User represents a user in the system who can organize or participate in meetings
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserResponse represents the response after creating a user
type CreateUserResponse struct {
	User User `json:"user"`
}

// GetUserResponse represents the response when fetching a user
type GetUserResponse struct {
	User User `json:"user"`
}

// ListUsersResponse represents the response when listing users
type ListUsersResponse struct {
	Users []User `json:"users"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}
