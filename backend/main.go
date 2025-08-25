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

	// Automatically start C2 listener profiles from config
	log.Println("🔒 Starting C2 listener profiles from configuration...")

	// Check if we have profiles configured
	if len(config.Profiles) == 0 {
		log.Fatalf("❌ FATAL: No C2 profiles configured in config.json")
	}

	// Check if at least one TLS profile is configured (this is a TLS-only system)
	tlsProfiles := 0
	for _, profileConfig := range config.Profiles {
		if profileConfig.UseTLS {
			tlsProfiles++
		}
	}

	if tlsProfiles == 0 {
		log.Fatalf("❌ FATAL: No TLS profiles configured. This system requires TLS encryption for all C2 communication.")
	}

	log.Printf("📋 Found %d profiles, %d with TLS enabled", len(config.Profiles), tlsProfiles)

	// Try to start each configured profile
	startedCount := 0
	criticalErrors := []string{}

	for _, profileConfig := range config.Profiles {
		log.Printf("🔄 Attempting to start profile: %s (%s:%d)", profileConfig.Name, profileConfig.Host, profileConfig.Port)
		log.Printf("📁 Profile config - UseTLS: %v, CertFile: %s, KeyFile: %s",
			profileConfig.UseTLS, profileConfig.CertFile, profileConfig.KeyFile)

		profile := &services.Profile{
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

		log.Printf("🔍 About to call StartListener for profile: %s", profile.Name)

		if err := listenerService.StartListener(profile); err != nil {
			errorMsg := fmt.Sprintf("❌ FAILED to start profile '%s': %v", profileConfig.Name, err)
			log.Printf(errorMsg)

			if profileConfig.UseTLS {
				log.Printf("💡 TLS is enabled but certificates are missing:")
				log.Printf("   - Cert: %s", profileConfig.CertFile)
				log.Printf("   - Key:  %s", profileConfig.KeyFile)

				// Collect critical errors for TLS profiles
				criticalErrors = append(criticalErrors, fmt.Sprintf("Profile '%s': %v", profileConfig.Name, err))
			}
		} else {
			log.Printf("✅ Profile '%s' started successfully on %s:%d", profileConfig.Name, profileConfig.Host, profileConfig.Port)
			if profileConfig.UseTLS {
				log.Printf("🔒 TLS 1.3/1.2 enabled - All C2 communication is encrypted")
			} else {
				log.Printf("⚠️  WARNING: Profile '%s' is NOT using TLS (plain TCP)", profileConfig.Name)
			}
			startedCount++
		}
	}

	// CRITICAL: If no listeners started or TLS profiles failed, exit
	if startedCount == 0 {
		log.Printf("")
		log.Printf("🚨 CRITICAL ERROR: NO C2 listeners were started successfully!")
		log.Printf("🚨 The server cannot function without active C2 listeners!")
		log.Printf("")

		if len(criticalErrors) > 0 {
			log.Printf("❌ TLS Certificate Errors:")
			for _, err := range criticalErrors {
				log.Printf("   - %s", err)
			}
			log.Printf("")
		}

		log.Printf("💡 To fix this issue:")
		log.Printf("   1. Generate TLS certificates: .\\generate-certs.ps1")
		log.Printf("   2. Ensure certificate files exist in the specified paths")
		log.Printf("   3. Check your config.json profile configuration")
		log.Printf("   4. Restart the server")
		log.Printf("")
		log.Printf("🚨 EXITING: Server cannot run without C2 listeners")
		os.Exit(1)
	}

	log.Printf("✅ Successfully started %d C2 listener(s)", startedCount)
	log.Printf("🔒 MuliC2 server is now ready with TLS encryption")

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
