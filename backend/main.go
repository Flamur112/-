package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"mulic2/handlers"
	"mulic2/services"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds the application configuration
type Config struct {
	Server struct {
		APIPort        int    `json:"api_port"`
		C2DefaultPort  int    `json:"c2_default_port"`
		TLSEnabled     bool   `json:"tls_enabled"`
		TLSMinVersion  string `json:"tls_min_version"`
		TLSMaxVersion  string `json:"tls_max_version"`
		APIUnified     bool   `json:"api_unified"`
		APIUnifiedPort int    `json:"api_unified_port"`
	} `json:"server"`
	Profiles []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ProjectName string `json:"projectName"`
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Description string `json:"description"`
		UseTLS      bool   `json:"useTLS"`
		CertFile    string `json:"certFile"`
		KeyFile     string `json:"keyFile"`
	} `json:"profiles"`
	Database struct {
		Type     string `json:"type"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
		SSLMode  string `json:"sslmode"`
	} `json:"database"`
	Logging struct {
		Level string `json:"level"`
	} `json:"logging"`
}

// Global VNC connections storage with proper synchronization
var (
	activeVNCConnections = make(map[string]*VNCConnection)
	vncMutex             sync.RWMutex
)

// VNCConnection represents a VNC connection
type VNCConnection struct {
	ConnectionID string    `json:"connection_id"`
	Hostname     string    `json:"hostname"`
	AgentIP      string    `json:"agent_ip"`
	Resolution   string    `json:"resolution"`
	FPS          int       `json:"fps"`
	ConnectedAt  time.Time `json:"connected_at"`
}

// Thread-safe VNC connection management
func addVNCConnection(id string, conn *VNCConnection) {
	vncMutex.Lock()
	defer vncMutex.Unlock()
	activeVNCConnections[id] = conn
}

func removeVNCConnection(id string) {
	vncMutex.Lock()
	defer vncMutex.Unlock()
	delete(activeVNCConnections, id)
}

func getVNCConnections() []VNCConnection {
	vncMutex.RLock()
	defer vncMutex.RUnlock()
	connections := make([]VNCConnection, 0, len(activeVNCConnections))
	for _, conn := range activeVNCConnections {
		connections = append(connections, *conn)
	}
	return connections
}

// loadConfig loads configuration from config.json
func loadConfig() (*Config, error) {
	configPaths := []string{
		"config.json",
		"../config.json",
		"./config.json",
	}

	var data []byte
	var err error

	for _, path := range configPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			log.Printf("Found config.json at: %s", path)
			break
		}
		log.Printf("Tried config path: %s - %v", path, err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read config file from any path: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// connectDB establishes database connection
func connectDB() (*sql.DB, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Password, config.Database.DBName, config.Database.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// createTables creates all necessary database tables
func createTables(db *sql.DB) error {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create agents table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id SERIAL PRIMARY KEY,
			agent_id VARCHAR(255) UNIQUE NOT NULL,
			hostname VARCHAR(255),
			ip_address VARCHAR(45),
			username VARCHAR(255),
			os_info VARCHAR(255),
			status VARCHAR(32) DEFAULT 'offline',
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}

	// Create tasks table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id SERIAL PRIMARY KEY,
			agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			command TEXT NOT NULL,
			status VARCHAR(32) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Create results table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			task_id INTEGER PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
			output TEXT,
			completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create results table: %w", err)
	}

	// Create profiles table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			id VARCHAR(128) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			project_name VARCHAR(255),
			host VARCHAR(64) DEFAULT '0.0.0.0',
			port INTEGER NOT NULL,
			description TEXT,
			use_tls BOOLEAN DEFAULT true,
			cert_file VARCHAR(512),
			key_file VARCHAR(512),
			poll_interval INTEGER DEFAULT 5,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create profiles table: %w", err)
	}

	return nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED: %v", r)
			log.Printf("Stack trace:")
			debug.PrintStack()
			log.Printf("Server crashed due to panic")
			time.Sleep(10 * time.Second)
		}
	}()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server config: API Port=%d, C2 Default Port=%d, TLS Enabled=%v",
		config.Server.APIPort, config.Server.C2DefaultPort, config.Server.TLSEnabled)

	// Database connection
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("Successfully connected to database")

	if err := createTables(db); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Initialize services
	listenerService := services.NewListenerService()
	defer listenerService.Close()
	profileStorage := services.NewProfileStorage(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	profileHandler := handlers.NewProfileHandler(db, listenerService)

	// Create main router
	router := mux.NewRouter()

	// Apply CORS middleware to the main router - THIS IS KEY!
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers for ALL requests
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Add logging middleware for debugging
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			next.ServeHTTP(w, r)
		})
	})

	// Root endpoint for testing
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Root request received from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":   "MuliC2 Backend is running!",
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}).Methods("GET", "OPTIONS")

	// Test CORS endpoint
	router.HandleFunc("/test-cors", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("CORS test request received from %s", r.RemoteAddr)
		log.Printf("Request headers: %v", r.Header)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "CORS is working!",
			"method":  r.Method,
			"origin":  r.Header.Get("Origin"),
			"path":    r.URL.Path,
		})
	}).Methods("GET", "POST", "OPTIONS")

	// Debug endpoint to test routing
	router.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Debug request received from %s", r.RemoteAddr)
		log.Printf("Request method: %s", r.Method)
		log.Printf("Request path: %s", r.URL.Path)
		log.Printf("Request headers: %v", r.Header)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Debug endpoint working",
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": r.Header,
			"remote":  r.RemoteAddr,
		})
	}).Methods("GET", "POST", "OPTIONS")

	// Create API subrouter
	api := router.PathPrefix("/api").Subrouter()

	// Apply CORS middleware to API subrouter as well
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers for ALL API requests
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health check endpoint - FIRST for testing
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Health check request received from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "MuliC2 Backend",
			"endpoints": []string{
				"/api/profile/list",
				"/api/profile/create",
				"/api/agents",
				"/api/tasks",
				"/api/vnc/start",
				"/api/vnc/stop",
			},
		})
	}).Methods("GET", "OPTIONS")

	// Simple test endpoint without database
	api.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Test endpoint request received from %s", r.RemoteAddr)
		log.Printf("Request headers: %v", r.Header)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Test endpoint working!",
			"method":    r.Method,
			"path":      r.URL.Path,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}).Methods("GET", "POST", "OPTIONS")

	// Profile endpoints - NO AUTH REQUIRED
	log.Printf("Registering profile endpoints...")

	// Profile creation endpoint
	api.HandleFunc("/profile/create", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Profile creation request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var profile struct {
			Name        string `json:"name"`
			ProjectName string `json:"projectName"`
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Description string `json:"description"`
			UseTLS      bool   `json:"useTLS"`
			CertFile    string `json:"certFile"`
			KeyFile     string `json:"keyFile"`
		}

		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		serviceProfile := &services.Profile{
			ID:          fmt.Sprintf("profile_%d", time.Now().Unix()),
			Name:        profile.Name,
			ProjectName: profile.ProjectName,
			Host:        profile.Host,
			Port:        profile.Port,
			Description: profile.Description,
			UseTLS:      profile.UseTLS,
			CertFile:    profile.CertFile,
			KeyFile:     profile.KeyFile,
		}

		// Try to start listener (don't fail if it can't)
		if err := listenerService.StartListener(serviceProfile); err != nil {
			log.Printf("Warning: Could not start listener for profile %s: %v", serviceProfile.ID, err)
		}

		// Save to database
		storedProfile := &services.StoredProfile{
			ID:           serviceProfile.ID,
			Name:         serviceProfile.Name,
			ProjectName:  serviceProfile.ProjectName,
			Host:         serviceProfile.Host,
			Port:         serviceProfile.Port,
			Description:  serviceProfile.Description,
			UseTLS:       serviceProfile.UseTLS,
			CertFile:     serviceProfile.CertFile,
			KeyFile:      serviceProfile.KeyFile,
			PollInterval: 5,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := profileStorage.SaveProfile(storedProfile); err != nil {
			log.Printf("Failed to save profile to database: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(serviceProfile)
		log.Printf("Profile created successfully: %s", serviceProfile.ID)
	}).Methods("POST", "OPTIONS")

	// Profile list endpoint
	api.HandleFunc("/profile/list", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Profile list request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("Request headers: %v", r.Header)
		log.Printf("Request method: %s", r.Method)

		if r.Method != "GET" {
			log.Printf("Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profiles, err := profileStorage.GetAllProfiles()
		if err != nil {
			log.Printf("Failed to get profiles: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get profiles: %v", err), http.StatusInternalServerError)
			return
		}

		profileData := make([]map[string]interface{}, 0, len(profiles))
		for _, profile := range profiles {
			profileInfo := map[string]interface{}{
				"id":           profile.ID,
				"name":         profile.Name,
				"projectName":  profile.ProjectName,
				"host":         profile.Host,
				"port":         profile.Port,
				"description":  profile.Description,
				"useTLS":       profile.UseTLS,
				"certFile":     profile.CertFile,
				"keyFile":      profile.KeyFile,
				"isActive":     profile.IsActive,
				"createdAt":    profile.CreatedAt,
				"updatedAt":    profile.UpdatedAt,
				"pollInterval": profile.PollInterval,
			}
			profileData = append(profileData, profileInfo)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"profiles": profileData,
		})
		log.Printf("Profile list returned: %d profiles", len(profileData))
	}).Methods("GET", "OPTIONS")

	// Profile delete endpoint
	api.HandleFunc("/profile/delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Profile delete request: %s %s", r.Method, r.URL.Path)

		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		vars := mux.Vars(r)
		profileID := vars["id"]
		if profileID == "" {
			http.Error(w, "Profile ID is required", http.StatusBadRequest)
			return
		}

		if err := profileStorage.DeleteProfile(profileID); err != nil {
			log.Printf("Failed to delete profile %s: %v", profileID, err)
			http.Error(w, fmt.Sprintf("Failed to delete profile: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Profile deleted successfully",
			"id":      profileID,
		})
		log.Printf("Profile deleted successfully: %s", profileID)
	}).Methods("DELETE", "OPTIONS")

	// Agents endpoint
	api.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Agents list request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents": []interface{}{},
		})
	}).Methods("GET", "OPTIONS")

	// Tasks endpoint
	api.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Tasks list request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": []interface{}{},
		})
	}).Methods("GET", "OPTIONS")

	// VNC endpoints
	api.HandleFunc("/vnc/start", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC start request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "VNC capture started",
			"status":  "active",
		})
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/vnc/stop", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC stop request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "VNC capture stopped",
			"status":  "inactive",
		})
	}).Methods("POST", "OPTIONS")

	// Register additional handler routes (for authenticated endpoints)
	authHandler.RegisterRoutes(api)
	profileHandler.RegisterRoutes(api)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.APIPort),
		Handler: router,
	}

	log.Printf("Starting HTTP server on port %d", config.Server.APIPort)
	log.Printf("Server will be accessible at:")
	log.Printf("  - http://localhost:%d", config.Server.APIPort)
	log.Printf("  - http://192.168.0.111:%d", config.Server.APIPort)

	go func() {
		log.Printf("HTTP server starting on :%d...", config.Server.APIPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Test server startup
	log.Printf("Waiting for server to start up...")
	for attempt := 1; attempt <= 10; attempt++ {
		time.Sleep(1 * time.Second)
		if resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", config.Server.APIPort)); err == nil {
			resp.Body.Close()
			log.Printf("HTTP server is ready and responding (attempt %d)", attempt)
			break
		}
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Server is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Printf("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("Server stopped successfully")
}
