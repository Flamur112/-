package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mulic2/services"
	"mulic2/utils"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/mux"
)

// ProfileHandler handles profile-related operations
type ProfileHandler struct {
	db              *sql.DB
	listenerService *services.ListenerService
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(db *sql.DB, listenerService *services.ListenerService) *ProfileHandler {
	return &ProfileHandler{
		db:              db,
		listenerService: listenerService,
	}
}

// RegisterRoutes registers the profile routes
func (h *ProfileHandler) RegisterRoutes(router *mux.Router) {
	// Protected routes - only authenticated users can manage profiles
	router.Handle("/profile/start", utils.AuthMiddleware(http.HandlerFunc(h.StartListener))).Methods("POST")
	router.Handle("/profile/stop", utils.AuthMiddleware(http.HandlerFunc(h.StopListener))).Methods("POST")
	router.Handle("/profile/status", utils.AuthMiddleware(http.HandlerFunc(h.GetStatus))).Methods("GET")
	router.Handle("/profile/list", utils.AuthMiddleware(http.HandlerFunc(h.ListProfiles))).Methods("GET")
	router.Handle("/profile/create", utils.AuthMiddleware(http.HandlerFunc(h.CreateProfile))).Methods("POST")
	router.Handle("/profile/get", utils.AuthMiddleware(http.HandlerFunc(h.GetProfile))).Methods("GET")
}

// StartListenerRequest represents the request to start a listener
type StartListenerRequest struct {
	Profile services.Profile `json:"profile"`
}

// validateHost validates the host field (IP address, localhost, or domain)
func (h *ProfileHandler) validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Check if it's localhost
	if host == "localhost" {
		return nil
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Check if it's a valid domain name
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if domainRegex.MatchString(host) {
		return nil
	}

	return fmt.Errorf("invalid host format: %s. Must be a valid IP address, 'localhost', or domain name", host)
}

// StartListener starts the C2 listener with the specified profile
func (h *ProfileHandler) StartListener(w http.ResponseWriter, r *http.Request) {
	var req StartListenerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate profile
	if req.Profile.Host == "" || req.Profile.Port <= 0 {
		http.Error(w, "Invalid profile configuration", http.StatusBadRequest)
		return
	}

	// Validate host format
	if err := h.validateHost(req.Profile.Host); err != nil {
		http.Error(w, "Invalid host: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate port range
	if req.Profile.Port < 1 || req.Profile.Port > 65535 {
		http.Error(w, "Invalid port: must be between 1 and 65535", http.StatusBadRequest)
		return
	}

	// Start the listener
	err := h.listenerService.StartListener(&req.Profile)
	if err != nil {
		http.Error(w, "Failed to start listener: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Listener started successfully",
		"profile": req.Profile,
	})
}

// GetStatus returns the current listener status
func (h *ProfileHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.listenerService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// StopListenerRequest represents a request to stop a specific listener
type StopListenerRequest struct {
	ProfileID string `json:"profileId"`
}

// StopListener stops a specific C2 listener by profile ID
func (h *ProfileHandler) StopListener(w http.ResponseWriter, r *http.Request) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in StopListener: %v", r)
			http.Error(w, "Internal server error during listener stop", http.StatusInternalServerError)
		}
	}()

	// Check if listener service is available
	if h.listenerService == nil {
		http.Error(w, "Listener service not available", http.StatusInternalServerError)
		return
	}

	// Parse request body
	var req StopListenerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate profile ID
	if req.ProfileID == "" {
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}

	err := h.listenerService.StopListener(req.ProfileID)
	if err != nil {
		log.Printf("Error stopping listener: %v", err)
		http.Error(w, "Failed to stop listener: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Listener stopped successfully",
		"profileId": req.ProfileID,
	})
}

// Profile management endpoints
func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, poll_interval, is_active, created_at, updated_at
		FROM profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("List profiles error: %v", err)
		http.Error(w, "Failed to list profiles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var profiles []map[string]interface{}
	for rows.Next() {
		var profile map[string]interface{}
		var id, name, projectName, host, description, certFile, keyFile string
		var port, pollInterval int
		var useTLS, isActive bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &name, &projectName, &host, &port, &description, &useTLS, &certFile, &keyFile, &pollInterval, &isActive, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Scan profile error: %v", err)
			continue
		}

		profile = map[string]interface{}{
			"id":           id,
			"name":         name,
			"projectName":  projectName,
			"host":         host,
			"port":         port,
			"description":  description,
			"useTLS":       useTLS,
			"certFile":     certFile,
			"keyFile":      keyFile,
			"pollInterval": pollInterval,
			"isActive":     isActive,
			"createdAt":    createdAt,
			"updatedAt":    updatedAt,
		}
		profiles = append(profiles, profile)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

type CreateProfileRequest struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProjectName  string `json:"projectName"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Description  string `json:"description"`
	UseTLS       bool   `json:"useTLS"`
	CertFile     string `json:"certFile"`
	KeyFile      string `json:"keyFile"`
	PollInterval int    `json:"pollInterval"`
}

func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enforce single-port C2: only allow host '0.0.0.0' and port 23456
	if req.Host != "0.0.0.0" || req.Port != 23456 {
		http.Error(w, "Only host '0.0.0.0' and port 23456 are allowed for single-port C2 operation", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ID == "" || req.Name == "" || req.Port <= 0 {
		http.Error(w, "ID, name, and port are required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Host == "" {
		req.Host = "0.0.0.0"
	}
	if req.PollInterval <= 0 {
		req.PollInterval = 5
	}

	_, err := h.db.Exec(`
		INSERT INTO profiles (id, name, project_name, host, port, description, use_tls, cert_file, key_file, poll_interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, req.ID, req.Name, req.ProjectName, req.Host, req.Port, req.Description, req.UseTLS, req.CertFile, req.KeyFile, req.PollInterval)
	if err != nil {
		log.Printf("Create profile error: %v", err)
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Profile created successfully",
		"profileId": req.ID,
	})
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("id")
	if profileID == "" {
		http.Error(w, "Profile ID required", http.StatusBadRequest)
		return
	}

	var profile map[string]interface{}
	var id, name, projectName, host, description, certFile, keyFile string
	var port, pollInterval int
	var useTLS, isActive bool
	var createdAt, updatedAt time.Time

	err := h.db.QueryRow(`
		SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, poll_interval, is_active, created_at, updated_at
		FROM profiles WHERE id = $1
	`, profileID).Scan(&id, &name, &projectName, &host, &port, &description, &useTLS, &certFile, &keyFile, &pollInterval, &isActive, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			log.Printf("Get profile error: %v", err)
			http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		}
		return
	}

	profile = map[string]interface{}{
		"id":           id,
		"name":         name,
		"projectName":  projectName,
		"host":         host,
		"port":         port,
		"description":  description,
		"useTLS":       useTLS,
		"certFile":     certFile,
		"keyFile":      keyFile,
		"pollInterval": pollInterval,
		"isActive":     isActive,
		"createdAt":    createdAt,
		"updatedAt":    updatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}
