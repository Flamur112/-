package models

import (
	"time"
)

// User represents a user account in the system
type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username" validate:"required,min=3,max=50,alphanum"`
	Email        *string    `json:"email,omitempty" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         string     `json:"role" db:"role" validate:"required,oneof=admin"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
}

// UserSession represents a user's active session
type UserSession struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"user_id" db:"user_id"`
	SessionToken string    `json:"session_token" db:"session_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	IsValid      bool      `json:"is_valid" db:"is_valid"`
}

// AuditLog represents security audit events
type AuditLog struct {
	ID        int       `json:"id" db:"id"`
	UserID    *int      `json:"user_id,omitempty" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Details   string    `json:"details" db:"details"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserSettings stores per-user listener preferences
type UserSettings struct {
	ID           int    `json:"id" db:"id"`
	UserID       int    `json:"user_id" db:"user_id"`
	ListenerIP   string `json:"listener_ip" db:"listener_ip"`
	ListenerPort int    `json:"listener_port" db:"listener_port"`
}

// ListenerProfile represents a named listener configuration
type ListenerProfile struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"user_id" db:"user_id"`
	Name         string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	ListenerIP   string    `json:"listener_ip" db:"listener_ip" validate:"required,ip"`
	ListenerPort int       `json:"listener_port" db:"listener_port" validate:"required,min=1024,max=65535"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ListenerProfileCreate represents data for creating a new profile
type ListenerProfileCreate struct {
	Name         string `json:"name" validate:"required,min=1,max=100"`
	ListenerIP   string `json:"listener_ip" validate:"required,ip"`
	ListenerPort int    `json:"listener_port" validate:"required,min=1024,max=65535"`
}

// ListenerProfileUpdate represents data for updating a profile
type ListenerProfileUpdate struct {
	Name         string `json:"name" validate:"required,min=1,max=100"`
	ListenerIP   string `json:"listener_ip" validate:"required,ip"`
	ListenerPort int    `json:"listener_port" validate:"required,min=1024,max=65535"`
	IsActive     bool   `json:"is_active"`
}

// UserRegistration represents the data needed to register a new user
type UserRegistration struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserLogin represents login credentials
type UserLogin struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UserResponse represents the user data sent to the frontend (without sensitive info)
type UserResponse struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

// IsAdmin checks if the user has admin privileges
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsOperator checks if the user has operator privileges
func (u *User) IsOperator() bool {
	return u.Role == "operator" || u.Role == "admin"
}

// CanView checks if the user has viewing privileges
func (u *User) CanView() bool {
	return u.IsActive
}
