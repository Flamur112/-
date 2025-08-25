package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mulic2/models"
	"mulic2/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type AuthHandler struct {
	db       *sql.DB
	validate *validator.Validate
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		db:       db,
		validate: validator.New(),
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegistration

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var exists int
	err := h.db.QueryRow("SELECT 1 FROM users WHERE username = $1", req.Username).Scan(&exists)
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	passwordHash := utils.HashPassword(req.Password, "")

	// Insert new user (force single role 'admin')
	result, err := h.db.Exec(`
		INSERT INTO users (username, password_hash, role, is_active)
		VALUES ($1, $2, 'admin', true)
	`, req.Username, passwordHash)

	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	userID, _ := result.LastInsertId()

	// Log the registration
	h.logAuditEvent(int(userID), "user_registered", fmt.Sprintf("New user registered: %s", req.Username), r)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User registered successfully",
		"user_id": userID,
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.UserLogin

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Get user from database
	var user models.User
	err := h.db.QueryRow(`
		SELECT id, username, email, password_hash, role, is_active
		FROM users WHERE username = $1
	`, req.Username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if account is active
	if !user.IsActive {
		http.Error(w, "Account is deactivated", http.StatusForbidden)
		return
	}

	// Verify password
	if !utils.VerifyPassword(req.Password, user.PasswordHash, "") {
		h.logAuditEvent(user.ID, "login_failed", "Invalid password", r)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Update last login
	h.db.Exec("UPDATE users SET last_login = NOW() WHERE id = $1", user.ID)

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Log successful login
	h.logAuditEvent(user.ID, "login_successful", "User logged in successfully", r)

	// Return token and user info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     "",
			Role:      "admin",
			IsActive:  user.IsActive,
			CreatedAt: time.Now(),
			LastLogin: nil,
		},
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := utils.ValidateJWT(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Extract user info
	userID, _, _, err := utils.ExtractUserFromJWT(claims)
	if err != nil {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	// Log the logout
	h.logAuditEvent(userID, "logout", "User logged out", r)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by middleware)
	userIDInterface := r.Context().Value("user_id")
	if userIDInterface == nil {
		http.Error(w, "Unauthorized - user not authenticated", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDInterface.(int)
	if !ok {
		http.Error(w, "Invalid user ID format", http.StatusInternalServerError)
		return
	}

	// Get user details from database
	var user models.User
	err := h.db.QueryRow(`
		SELECT id, username, email, role, is_active, created_at, last_login
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.IsActive, &user.CreatedAt, &user.LastLogin)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Return user profile
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     "",
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		LastLogin: user.LastLogin,
	})
}

// logAuditEvent logs security events to the audit log
func (h *AuthHandler) logAuditEvent(userID int, action, details string, r *http.Request) {
	ipAddress := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ipAddress = forwardedFor
	}

	userAgent := r.Header.Get("User-Agent")

	_, err := h.db.Exec(`
		INSERT INTO audit_logs (user_id, action, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, userID, action, details, ipAddress, userAgent)

	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to log audit event: %v\n", err)
	}
}

// RegisterRoutes registers the authentication routes
func (h *AuthHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/auth/register", h.Register).Methods("POST")
	router.HandleFunc("/auth/login", h.Login).Methods("POST")
	router.HandleFunc("/auth/logout", h.Logout).Methods("POST")

	// Protected routes
	router.Handle("/auth/profile", utils.AuthMiddleware(http.HandlerFunc(h.GetProfile))).Methods("GET")
	router.Handle("/settings/listener", utils.AuthMiddleware(http.HandlerFunc(h.GetListenerSettings))).Methods("GET")
	router.Handle("/settings/listener", utils.AuthMiddleware(http.HandlerFunc(h.UpdateListenerSettings))).Methods("PUT")
}

// GetListenerSettings returns the current user's listener settings
func (h *AuthHandler) GetListenerSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var settings models.UserSettings
	err := h.db.QueryRow(`
        SELECT id, user_id, listener_ip, listener_port FROM user_settings WHERE user_id = $1
    `, userID).Scan(&settings.ID, &settings.UserID, &settings.ListenerIP, &settings.ListenerPort)

	if err == sql.ErrNoRows {
		settings = models.UserSettings{UserID: userID, ListenerIP: "0.0.0.0", ListenerPort: 8080}
	} else if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// UpdateListenerSettings upserts listener settings
func (h *AuthHandler) UpdateListenerSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var body struct {
		ListenerIP   string `json:"listener_ip"`
		ListenerPort int    `json:"listener_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if body.ListenerIP == "" {
		body.ListenerIP = "0.0.0.0"
	}
	if body.ListenerPort <= 0 || body.ListenerPort > 65535 {
		body.ListenerPort = 8080
	}

	_, err := h.db.Exec(`
        INSERT INTO user_settings (user_id, listener_ip, listener_port)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id) DO UPDATE SET listener_ip = EXCLUDED.listener_ip, listener_port = EXCLUDED.listener_port
    `, userID, body.ListenerIP, body.ListenerPort)
	if err != nil {
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"message": "saved"})
}
