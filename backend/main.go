package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
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

// loadConfig loads configuration from config.json
func loadConfig() (*Config, error) {
	configFile := "config.json"

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		defaultConfig := &Config{}
		defaultConfig.Server.APIPort = 8080
		defaultConfig.Server.C2DefaultPort = 8081
		defaultConfig.Database.Type = "postgres"
		defaultConfig.Database.Host = "localhost"
		defaultConfig.Database.Port = 5432
		defaultConfig.Database.User = "postgres"
		defaultConfig.Database.Password = "postgres"
		defaultConfig.Database.DBName = "mulic2_db"
		defaultConfig.Database.SSLMode = "disable"
		defaultConfig.Logging.Level = "info"

		// Save default config
		configData, _ := json.MarshalIndent(defaultConfig, "", "  ")
		os.WriteFile(configFile, configData, 0644)

		log.Printf("Created default config.json with ports: API=%d, C2=%d",
			defaultConfig.Server.APIPort, defaultConfig.Server.C2DefaultPort)

		return defaultConfig, nil
	}

	// Read existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
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

	// Initialize listener service
	listenerService := services.NewListenerService()
	defer listenerService.Close()

	// Initialize listener storage
	listenerStorage := services.NewListenerStorage(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	profileHandler := handlers.NewProfileHandler(db, listenerService)
	agentHandler := handlers.NewAgentHandler(db)
	operatorHandler := handlers.NewOperatorHandler(db)

	// Create router
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()

	// CORS middleware
	router.Use(corsMiddleware)

	// Register routes under /api
	authHandler.RegisterRoutes(api)
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

	// Listener management endpoints (protected)
	api.Handle("/listeners", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listeners, err := listenerStorage.GetAllListeners()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get listeners: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"listeners": listeners,
		})
	}))).Methods("GET")

	api.Handle("/listeners", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var listener services.StoredListener
		if err := json.NewDecoder(r.Body).Decode(&listener); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Generate unique ID if not provided
		if listener.ID == "" {
			listener.ID = fmt.Sprintf("listener_%d", time.Now().Unix())
		}

		// Set timestamps
		if listener.CreatedAt.IsZero() {
			listener.CreatedAt = time.Now()
		}
		listener.UpdatedAt = time.Now()

		if err := listenerStorage.SaveListener(&listener); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save listener: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(listener)
	}))).Methods("POST")

	api.Handle("/listeners/{id}/start", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		listenerID := vars["id"]

		// Get listener from storage
		listener, err := listenerStorage.GetListener(listenerID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Listener not found: %v", err), http.StatusNotFound)
			return
		}

		// Start the listener
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
			http.Error(w, fmt.Sprintf("Failed to start listener: %v", err), http.StatusInternalServerError)
			return
		}

		// Mark as active in storage
		if err := listenerStorage.UpdateListenerStatus(listenerID, true); err != nil {
			log.Printf("Warning: Failed to update listener status: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "Listener started successfully",
			"listener": listener,
		})
	}))).Methods("POST")

	api.Handle("/listeners/{id}/stop", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		listenerID := vars["id"]

		// Stop the listener
		if err := listenerService.StopListener(listenerID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop listener: %v", err), http.StatusInternalServerError)
			return
		}

		// Mark as inactive in storage
		if err := listenerStorage.UpdateListenerStatus(listenerID, false); err != nil {
			log.Printf("Warning: Failed to update listener status: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Listener stopped successfully",
		})
	}))).Methods("POST")

	api.Handle("/listeners/{id}", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		listenerID := vars["id"]

		if err := listenerStorage.DeleteListener(listenerID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete listener: %v", err), http.StatusInternalServerError)
			return
		}

		// Also stop the listener if it's running
		listenerService.StopListener(listenerID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Listener deleted successfully",
		})
	}))).Methods("DELETE")

	// VNC endpoints (protected)
	api.Handle("/vnc/connections", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connections := listenerService.GetVNCService().GetActiveConnections()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connections": connections,
		})
	}))).Methods("GET")

	// VNC frame streaming endpoint (Server-Sent Events)
	api.Handle("/vnc/stream", utils.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get VNC service
		vncService := listenerService.GetVNCService()
		frameChannel := vncService.GetFrameChannel()

		// Create a channel to detect client disconnect
		notify := w.(http.CloseNotifier).CloseNotify()

		for {
			select {
			case frame := <-frameChannel:
				// Send frame data as SSE
				frameData := map[string]interface{}{
					"connection_id": frame.ConnectionID,
					"timestamp":     frame.Timestamp,
					"width":         frame.Width,
					"height":        frame.Height,
					"data":          frame.Data,
					"size":          frame.Size,
				}

				frameJSON, _ := json.Marshal(frameData)
				fmt.Fprintf(w, "data: %s\n\n", frameJSON)
				w.(http.Flusher).Flush()

			case <-notify:
				// Client disconnected
				return
			}
		}
	}))).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "mulic2"}`))
	}).Methods("GET")

	// Protected routes (example)
	protected := router.PathPrefix("/api/protected").Subrouter()
	protected.Use(utils.AuthMiddleware)
	protected.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Access granted to protected resource"}`))
	}).Methods("GET")

	// Set router for unified API mode
	if config.Server.APIUnified {
		listenerService.SetRouter(router)
		log.Printf("🔗 Router configured for unified API mode")
	}

	// Start server
	apiPort := fmt.Sprintf("%d", config.Server.APIPort)
	c2Port := fmt.Sprintf("%d", config.Server.C2DefaultPort)

	// Validate ports before starting
	if config.Server.APIPort <= 0 || config.Server.APIPort > 65535 {
		log.Fatalf("Invalid API port: %d. Port must be between 1 and 65535", config.Server.APIPort)
	}
	if config.Server.C2DefaultPort <= 0 || config.Server.C2DefaultPort > 65535 {
		log.Fatalf("Invalid C2 default port: %d. Port must be between 1 and 65535", config.Server.C2DefaultPort)
	}

	// Check unified API mode
	if config.Server.APIUnified {
		log.Printf("🔗 UNIFIED MODE: API will be served through TLS listener on port %d", config.Server.APIUnifiedPort)
		if config.Server.APIPort == config.Server.APIUnifiedPort {
			log.Printf("⚠️  WARNING: API port and unified port are the same - API will be served through TLS")
		}
	} else {
		if config.Server.APIPort == config.Server.C2DefaultPort {
			log.Fatalf("API port (%d) and C2 default port (%d) cannot be the same in separated mode", config.Server.APIPort, config.Server.C2DefaultPort)
		}
		log.Printf("🔌 SEPARATED MODE: API on port %s, C2 listeners on separate ports", apiPort)
	}

	log.Printf("Starting MuliC2 server on port %s", apiPort)
	log.Printf("C2 listeners will use port %s by default", c2Port)
	log.Printf("⚠️  IMPORTANT: C2 listeners will ONLY use the exact ports specified in profiles - NO FALLBACK PORTS!")

	// Test if API port is available (only in separated mode)
	if !config.Server.APIUnified {
		testListener, err := net.Listen("tcp", ":"+apiPort)
		if err != nil {
			log.Fatalf("❌ FAILED: Cannot bind to API port %s: %v", apiPort, err)
		}
		testListener.Close()
		log.Printf("✅ API port %s is available", apiPort)
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine (only in separated mode)
	if !config.Server.APIUnified {
		go func() {
			log.Printf("Server starting on port %s...", apiPort)
			addr := ":" + apiPort
			log.Printf("Binding to address: %s", addr)
			if err := http.ListenAndServe(addr, router); err != nil {
				log.Printf("Server error: %v", err)
			}
		}()

		// Wait a moment for server to start
		time.Sleep(2 * time.Second)
		log.Println("Server is ready and listening")
	}

	// Initialize listener storage and load default profiles
	log.Println("🔒 Initializing listener management system...")

	if err := listenerStorage.Initialize(); err != nil {
		log.Fatalf("❌ Failed to initialize listener storage: %v", err)
	}

	// Load default profiles from config.json into database (but don't start them)
	// Convert config profiles to services.Profile type
	var defaultProfiles []services.Profile
	for _, profileConfig := range config.Profiles {
		profile := services.Profile{
			ID:          profileConfig.ID,
			Name:        profileConfig.Name,
			ProjectName: profileConfig.ProjectName,
			Host:        profileConfig.Host,
			Port:        profileConfig.Port,
			Description: profileConfig.Description,
			UseTLS:      profileConfig.UseTLS,
			CertFile:    profileConfig.CertFile,
			KeyFile:     profileConfig.KeyFile,
		}
		defaultProfiles = append(defaultProfiles, profile)
	}

	if err := listenerStorage.LoadDefaultListeners(defaultProfiles); err != nil {
		log.Printf("⚠️  Warning: Failed to load default listeners: %v", err)
	}

	// Only start listeners that are explicitly marked as active in the database
	log.Println("🔒 Starting active C2 listeners from database...")

	activeListeners, err := listenerStorage.GetActiveListeners()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to get active listeners: %v", err)
		activeListeners = []*services.StoredListener{} // Empty slice
	}

	if len(activeListeners) == 0 {
		log.Printf("💡 No active listeners found. Listeners must be manually started from the dashboard.")
		log.Printf("💡 Default profiles are loaded but inactive. Use the dashboard to start them.")
	} else {
		log.Printf("📋 Found %d active listeners", len(activeListeners))

		// Start each active listener
		startedCount := 0
		for _, storedListener := range activeListeners {
			log.Printf("🔄 Starting active listener: %s (%s:%d)", storedListener.Name, storedListener.Host, storedListener.Port)

			// Convert StoredListener to Profile
			profile := &services.Profile{
				ID:          storedListener.ID,
				Name:        storedListener.Name,
				ProjectName: storedListener.ProjectName,
				Host:        storedListener.Host,
				Port:        storedListener.Port,
				Description: storedListener.Description,
				UseTLS:      storedListener.UseTLS,
				CertFile:    storedListener.CertFile,
				KeyFile:     storedListener.KeyFile,
			}

			if err := listenerService.StartListener(profile); err != nil {
				log.Printf("❌ Failed to start listener '%s': %v", storedListener.Name, err)
				// Mark as inactive in database since it failed to start
				listenerStorage.UpdateListenerStatus(storedListener.ID, false)
			} else {
				log.Printf("✅ Listener '%s' started successfully on %s:%d", storedListener.Name, storedListener.Host, storedListener.Port)
				if storedListener.UseTLS {
					log.Printf("🔒 TLS 1.3/1.2 enabled - All C2 communication is encrypted")
				}
				startedCount++
			}
		}

		log.Printf("✅ Successfully started %d active listener(s)", startedCount)
	}

	log.Printf("🔒 MuliC2 server is ready. Use the dashboard to manage listeners.")

	// Start background agent status monitoring
	go monitorAgentStatus(db)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down server...")

	// Stop listener service
	if err := listenerService.StopAllListeners(); err != nil {
		log.Printf("Error stopping listeners: %v", err)
	}

	log.Println("Server stopped")
}

// monitorAgentStatus runs in the background to mark agents as offline if they haven't been seen recently
func monitorAgentStatus(db *sql.DB) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Mark agents as offline if they haven't been seen in the last 2 minutes
			result, err := db.Exec(`
				UPDATE agents 
				SET status = 'offline' 
				WHERE status = 'online' 
				AND last_seen < NOW() - INTERVAL '2 minutes'
			`)
			if err != nil {
				log.Printf("Error monitoring agent status: %v", err)
				continue
			}

			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				log.Printf("🔄 Marked %d agents as offline due to inactivity", rowsAffected)
			}
		}
	}
}

func connectDB() (*sql.DB, error) {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// PostgreSQL connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.DBName,
		config.Database.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize default profiles from config
	if err := initializeProfiles(db, config); err != nil {
		return nil, fmt.Errorf("failed to initialize profiles: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	return db, nil
}

func initializeProfiles(db *sql.DB, config *Config) error {
	// Check if profiles table has any data
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check profiles count: %w", err)
	}

	// If profiles exist, don't initialize
	if count > 0 {
		return nil
	}

	// Insert default profiles from config
	for _, profile := range config.Profiles {
		_, err := db.Exec(`
			INSERT INTO profiles (id, name, project_name, host, port, description, use_tls, cert_file, key_file, poll_interval)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO NOTHING
		`, profile.ID, profile.Name, profile.ProjectName, profile.Host, profile.Port, profile.Description, profile.UseTLS, profile.CertFile, profile.KeyFile, 5)
		if err != nil {
			log.Printf("Warning: Failed to insert profile %s: %v", profile.ID, err)
		}
	}

	log.Printf("✅ Initialized %d default profiles in database", len(config.Profiles))
	return nil
}

func createTables(db *sql.DB) error {
	// Create users table with PostgreSQL syntax
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255),
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Agents table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id SERIAL PRIMARY KEY,
			hostname VARCHAR(255),
			username VARCHAR(255),
			ip VARCHAR(64),
			os VARCHAR(128),
			arch VARCHAR(64),
			profile_id VARCHAR(128),
			status VARCHAR(32) DEFAULT 'online',
			first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}

	// Tasks table
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

	// Results table
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

	// Profiles table
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

	// Create user_settings table with PostgreSQL syntax
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

	// Create audit_logs table with PostgreSQL syntax
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
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
