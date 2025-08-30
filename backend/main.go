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
	"mulic2/utils"

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

// VNCConnection represents a VNC connection
type VNCConnection struct {
	ConnectionID string    `json:"connection_id"`
	Hostname     string    `json:"hostname"`
	AgentIP      string    `json:"agent_ip"`
	Resolution   string    `json:"resolution"`
	FPS          int       `json:"fps"`
	ConnectedAt  time.Time `json:"connected_at"`
}

// Global VNC connections storage with proper synchronization
var (
	activeVNCConnections = make(map[string]*VNCConnection)
	vncMutex             sync.RWMutex
)

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
	// Try multiple paths for config.json
	configPaths := []string{
		"config.json",    // Current directory
		"../config.json", // Parent directory
		"./config.json",  // Explicit current directory
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

	// Test connection
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

	// Create user_settings table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_settings (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			listener_ip VARCHAR(45) DEFAULT '0.0.0.0',
			listener_port INTEGER DEFAULT 8080,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_settings table: %w", err)
	}

	// Create audit_logs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			action VARCHAR(100) NOT NULL,
			details TEXT,
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 PANIC RECOVERED: %v", r)
			log.Printf("🚨 Stack trace:")
			debug.PrintStack()
			log.Printf("🚨 Server crashed due to panic")
			time.Sleep(10 * time.Second) // Keep window open
		}
	}()

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server config: API Port=%d, C2 Default Port=%d, TLS Enabled=%v",
		config.Server.APIPort, config.Server.C2DefaultPort, config.Server.TLSEnabled)
	log.Printf("Profiles loaded: %d", len(config.Profiles))

	// Database connection
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("Successfully connected to database")

	// Create tables
	if err := createTables(db); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Initialize listener service
	listenerService := services.NewListenerService()
	defer listenerService.Close()

	// Load and start profiles from config.json
	log.Printf("Loading profiles from config.json...")
	log.Printf("Total profiles found: %d", len(config.Profiles))

	if len(config.Profiles) == 0 {
		log.Printf("⚠️  WARNING: No profiles found in config.json!")
	} else {
		for i, profile := range config.Profiles {
			log.Printf("Profile %d: %s (Port: %d, TLS: %v, Cert: %s, Key: %s)",
				i+1, profile.Name, profile.Port, profile.UseTLS, profile.CertFile, profile.KeyFile)

			// Convert config profile to service profile
			serviceProfile := &services.Profile{
				ID:          profile.ID,
				Name:        profile.Name,
				ProjectName: profile.ProjectName,
				Host:        profile.Host,
				Port:        profile.Port,
				Description: profile.Description,
				UseTLS:      profile.UseTLS,
				CertFile:    profile.CertFile,
				KeyFile:     profile.KeyFile,
			}

			log.Printf("Starting listener for profile: %s...", profile.Name)
			if err := listenerService.StartListener(serviceProfile); err != nil {
				log.Printf("❌ Failed to start listener for profile %s: %v", profile.Name, err)
			} else {
				log.Printf("✅ Successfully started listener for profile %s", profile.Name)
			}
		}
	}

	// Also ensure the default profile from config.json is always started
	log.Printf("Ensuring default profile is running...")

	// Create default profile if none exists
	if len(config.Profiles) == 0 {
		log.Printf("No profiles found in config.json, creating default profile...")
		defaultProfile := &services.Profile{
			ID:          "default",
			Name:        "Default C2",
			ProjectName: "MuliC2",
			Host:        "0.0.0.0",
			Port:        23456,
			Description: "Default C2 profile with TLS enabled",
			UseTLS:      true,
			CertFile:    "../server.crt",
			KeyFile:     "../server.key",
		}

		if err := listenerService.StartListener(defaultProfile); err != nil {
			log.Printf("❌ Failed to start default listener: %v", err)
		} else {
			log.Printf("✅ Default profile started successfully")
		}
	}

	// Initialize storage services
	listenerStorage := services.NewListenerStorage(db)
	profileStorage := services.NewProfileStorage(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	profileHandler := handlers.NewProfileHandler(db, listenerService)
	agentHandler := handlers.NewAgentHandler(db)
	operatorHandler := handlers.NewOperatorHandler(db)

	// Create router
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()

	// CORS middleware - apply to both main router and API subrouter
	router.Use(corsMiddleware)
	api.Use(corsMiddleware)

	// Root endpoint for testing
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 Root request received")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "MuliC2 Backend is running!",
			"status":  "ok",
		})
	}).Methods("GET")

	// Register routes under /api
	authHandler.RegisterRoutes(api)

	// Profile creation endpoint (for frontend auto-creation) - NO AUTH REQUIRED
	// MUST BE REGISTERED BEFORE profileHandler to override the protected route
	log.Printf("🔧 Registering /api/profile/create endpoint (no auth required)...")
	api.HandleFunc("/profile/create", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 Received profile creation request: %s %s", r.Method, r.URL.Path)

		if r.Method != "POST" {
			log.Printf("❌ Method not allowed: %s", r.Method)
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

		// Create service profile
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

		// Start the listener (but don't fail if it can't start due to port conflict)
		if err := listenerService.StartListener(serviceProfile); err != nil {
			log.Printf("⚠️  Warning: Could not start listener for profile %s: %v", serviceProfile.ID, err)
			// Don't fail the request - just log the warning and continue
		} else {
			log.Printf("✅ Listener started successfully for profile %s", serviceProfile.ID)
		}

		// Save profile to database
		storedProfile := &services.StoredProfile{
			ID:          serviceProfile.ID,
			Name:        serviceProfile.Name,
			ProjectName: serviceProfile.ProjectName,
			Host:        serviceProfile.Host,
			Port:        serviceProfile.Port,
			Description: serviceProfile.Description,
			UseTLS:      serviceProfile.UseTLS,
			CertFile:    serviceProfile.CertFile,
			KeyFile:     serviceProfile.KeyFile,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := profileStorage.SaveProfile(storedProfile); err != nil {
			log.Printf("⚠️  Warning: Could not save profile to database: %v", err)
			// Don't fail the request - just log the warning and continue
		} else {
			log.Printf("✅ Profile saved to database: %s", serviceProfile.ID)
		}

		// Return the created profile
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(serviceProfile)
		log.Printf("✅ Profile created successfully: %s", serviceProfile.ID)
	}).Methods("POST")

	// Profile list endpoint (for frontend dashboard) - NO AUTH REQUIRED
	log.Printf("🔧 Registering /api/profile/list endpoint (no auth required)...")
	api.HandleFunc("/profile/list", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 Received profile list request: %s %s", r.Method, r.URL.Path)

		if r.Method != "GET" {
			log.Printf("❌ Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get all profiles from storage
		profiles, err := profileStorage.GetAllProfiles()
		if err != nil {
			log.Printf("❌ Failed to get profiles: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get profiles: %v", err), http.StatusInternalServerError)
			return
		}

		// Convert profiles to response format
		profileData := make([]map[string]interface{}, 0, len(profiles))
		for _, profile := range profiles {
			profileInfo := map[string]interface{}{
				"id":          profile.ID,
				"name":        profile.Name,
				"projectName": profile.ProjectName,
				"host":        profile.Host,
				"port":        profile.Port,
				"description": profile.Description,
				"useTLS":      profile.UseTLS,
				"certFile":    profile.CertFile,
				"keyFile":     profile.KeyFile,
				"isActive":    profile.IsActive,
				"createdAt":   profile.CreatedAt,
				"updatedAt":   profile.UpdatedAt,
			}
			profileData = append(profileData, profileInfo)
		}

		// Return the profiles
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"profiles": profileData,
		})
		log.Printf("✅ Profile list returned: %d profiles", len(profileData))
	}).Methods("GET")

	// Register profile handler routes AFTER our custom route
	profileHandler.RegisterRoutes(api)

	// Agent routes (REST)
	api.HandleFunc("/agent/register", agentHandler.Register).Methods("POST")
	api.HandleFunc("/agent/heartbeat", agentHandler.Heartbeat).Methods("POST")
	api.HandleFunc("/agent/tasks", agentHandler.FetchTasks).Methods("GET")
	api.HandleFunc("/agent/result", agentHandler.SubmitResult).Methods("POST")

	// Operator endpoints (protected)
	api.Handle("/agents", utils.AuthMiddleware(http.HandlerFunc(operatorHandler.ListAgents))).Methods("GET")
	api.Handle("/tasks", utils.AuthMiddleware(http.HandlerFunc(operatorHandler.EnqueueTask))).Methods("POST")
	api.Handle("/agent-tasks", utils.AuthMiddleware(http.HandlerFunc(operatorHandler.GetAgentTasks))).Methods("GET")

	// Health check endpoint
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 Health check request received")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "MuliC2 Backend",
		})
	}).Methods("GET")

	// Simple test endpoint
	api.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 Test endpoint request received")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Test endpoint working!",
		})
	}).Methods("GET")

	// Agent template download endpoint (protected)
	api.Handle("/agent/template", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=vnc_agent_template.ps1")

		// Read and serve the updated agent template
		templatePath := "../frontend/src/utils/vnc_agent_template.ps1"
		http.ServeFile(w, r, templatePath)
	}))).Methods("GET")

	// Listener management endpoints (protected)
	api.Handle("/listeners", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// List all listeners
			listeners, err := listenerStorage.GetAllListeners()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get listeners: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"listeners": listeners,
			})
		case "POST":
			// Create new listener
			var listener services.StoredListener
			if err := json.NewDecoder(r.Body).Decode(&listener); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			listener.ID = fmt.Sprintf("listener_%d", time.Now().Unix())
			listener.CreatedAt = time.Now()
			listener.UpdatedAt = time.Now()

			// Set default values for TLS fields if not provided
			if listener.CertFile == "" {
				listener.CertFile = "../server.crt"
			}
			if listener.KeyFile == "" {
				listener.KeyFile = "../server.key"
			}
			// Default to TLS enabled for security
			if !listener.UseTLS {
				listener.UseTLS = true
			}

			if err := listenerStorage.SaveListener(&listener); err != nil {
				http.Error(w, fmt.Sprintf("Failed to save listener: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(listener)
		}
	}))).Methods("GET", "POST")

	// Listener start/stop endpoints
	api.Handle("/listeners/{id}/start", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Update listener status to active
		if err := listenerStorage.UpdateListenerStatus(id, true); err != nil {
			http.Error(w, fmt.Sprintf("Failed to start listener: %v", err), http.StatusInternalServerError)
			return
		}

		// Get listener details and start it
		listener, err := listenerStorage.GetListener(id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Listener not found: %v", err), http.StatusNotFound)
			return
		}

		// Convert to Profile and start
		profile := &services.Profile{
			ID:          listener.ID,
			Name:        listener.Name,
			ProjectName: listener.ProjectName,
			Host:        listener.Host,
			Port:        listener.Port,
			Description: listener.Description,
			UseTLS:      listener.UseTLS,
			CertFile:    listener.CertFile,
			KeyFile:     listener.KeyFile,
		}

		if err := listenerService.StartListener(profile); err != nil {
			// Revert status if failed to start
			listenerStorage.UpdateListenerStatus(id, false)
			http.Error(w, fmt.Sprintf("Failed to start listener: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	}))).Methods("POST")

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.APIPort),
		Handler: router,
	}

	log.Printf("Starting HTTP server on port %d", config.Server.APIPort)
	log.Printf("Router configured with API subrouter")

	// Start server in background goroutine with better error handling
	go func() {
		log.Printf("🔄 HTTP server starting on :%d...", config.Server.APIPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ HTTP server error: %v", err)
			// If there's a binding error, log it clearly
			if err.Error() == "bind: address already in use" {
				log.Printf("❌ Port %d is already in use! Please check if another process is using this port.", config.Server.APIPort)
			}
		} else {
			log.Printf("✅ HTTP server stopped normally")
		}
	}()

	// Give the server time to start up and test multiple times
	log.Printf("⏳ Waiting for server to start up...")

	// Test server startup with multiple attempts
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(1 * time.Second)

		log.Printf("🔄 Testing server response (attempt %d/%d)...", attempt, maxAttempts)

		// Test root endpoint first
		if resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", config.Server.APIPort)); err == nil {
			resp.Body.Close()
			log.Printf("✅ Root endpoint working (attempt %d)", attempt)
		} else {
			log.Printf("⚠️  Root endpoint failed: %v", err)
		}

		// Test health endpoint
		if resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", config.Server.APIPort)); err == nil {
			resp.Body.Close()
			log.Printf("✅ HTTP server is ready and responding (attempt %d)", attempt)
			break
		} else {
			log.Printf("⚠️  Health endpoint failed: %v", err)
			if attempt == maxAttempts {
				log.Printf("❌ HTTP server failed to respond after %d attempts", maxAttempts)
				log.Printf("❌ This may indicate a port conflict or server startup issue")
			}
		}
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("🚀 Server is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Printf("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Stop all listeners
	if err := listenerService.StopAllListeners(); err != nil {
		log.Printf("Error stopping listeners: %v", err)
	}

	log.Printf("Server stopped successfully")
}
