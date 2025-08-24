package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mulic2/services"
	"net"
	"net/http"
	"regexp"

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
	router.HandleFunc("/profile/start", h.StartListener).Methods("POST")
	router.HandleFunc("/profile/stop", h.StopListener).Methods("POST")
	router.HandleFunc("/profile/status", h.GetStatus).Methods("GET")
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

// StopListener stops the current C2 listener
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

	err := h.listenerService.StopListener()
	if err != nil {
		log.Printf("Error stopping listener: %v", err)
		http.Error(w, "Failed to stop listener: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Listener stopped successfully",
	})
}
