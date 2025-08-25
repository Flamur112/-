package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os" // Added for os.Stat
	"strings"
	"sync"
	"time"
)

// Profile represents a server profile configuration
type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Description string `json:"description"`
	UseTLS      bool   `json:"useTLS"`
	CertFile    string `json:"certFile"`
	KeyFile     string `json:"keyFile"`
}

// ListenerService manages C2 server listeners
type ListenerService struct {
	mu        sync.RWMutex
	listeners map[string]*listenerInstance // Map of profile ID to listener instance
	ctx       context.Context
	cancel    context.CancelFunc
}

// listenerInstance represents a single listener instance
type listenerInstance struct {
	listener net.Listener
	profile  *Profile
	active   bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewListenerService creates a new listener service
func NewListenerService() *ListenerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ListenerService{
		listeners: make(map[string]*listenerInstance),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// createTLSConfig creates TLS configuration with TLS 1.3 support
func (ls *ListenerService) createTLSConfig(profile *Profile) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	// TLS requires certificate files - no fallback to self-signed
	if profile.CertFile == "" || profile.KeyFile == "" {
		return nil, fmt.Errorf("TLS is enabled but certificate files are not specified. Please provide CertFile and KeyFile in profile configuration")
	}

	// Load user-provided certificates
	cert, err = tls.LoadX509KeyPair(profile.CertFile, profile.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate files: %w", err)
	}
	log.Printf("🔒 Loaded user certificate from %s and %s", profile.CertFile, profile.KeyFile)

	// Create TLS config with modern security settings
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Minimum TLS 1.2
		MaxVersion:   tls.VersionTLS13, // Maximum TLS 1.3
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (preferred)
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			// TLS 1.2 fallback cipher suites
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		// Allow client to choose TLS version (1.3 or 1.2)
		ClientAuth: tls.NoClientCert,
	}

	return tlsConfig, nil
}

// StartListener starts a new C2 listener with the specified profile
func (ls *ListenerService) StartListener(profile *Profile) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Validate profile
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}

	// Check if this specific profile is already active
	if existing, exists := ls.listeners[profile.ID]; exists && existing.active {
		return fmt.Errorf("listener for profile '%s' (ID: %s) is already active", profile.Name, profile.ID)
	}

	// Check if port is privileged (requires root or setcap)
	if profile.Port < 1024 {
		log.Printf("⚠️  WARNING: Port %d is privileged (< 1024). Ensure the backend has proper permissions:", profile.Port)
		log.Printf("   - Run as root: sudo ./mulic2")
		log.Printf("   - Or apply setcap: sudo setcap 'cap_net_bind_service=+ep' ./mulic2")
		log.Printf("   - Or use a non-privileged port (>= 1024)")
	}

	// Check port availability first
	if err := ls.checkPortAvailability(profile.Host, profile.Port); err != nil {
		return fmt.Errorf("port %d is not available: %w", profile.Port, err)
	}

	addr := fmt.Sprintf("%s:%d", profile.Host, profile.Port)
	var listener net.Listener
	var err error

	if profile.UseTLS {
		// Validate certificate files exist when TLS is enabled
		if profile.CertFile == "" || profile.KeyFile == "" {
			return fmt.Errorf("TLS is enabled but certificate files are not specified. Please provide CertFile and KeyFile in profile configuration")
		}

		// Check if certificate files exist
		if _, err := os.Stat(profile.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file not found: %s", profile.CertFile)
		}
		if _, err := os.Stat(profile.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("private key file not found: %s", profile.KeyFile)
		}

		// Load TLS configuration
		tlsConfig, err := ls.createTLSConfig(profile)
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %w", err)
		}

		// Create TCP listener first
		tcpListener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to start TCP listener on %s: %w", addr, err)
		}

		// Wrap with TLS listener
		listener = tls.NewListener(tcpListener, tlsConfig)
		log.Printf("🔒 TLS C2 Listener started on %s (Profile: %s) - TLS 1.3/1.2 enabled with certificates", addr, profile.Name)
	} else {
		// Create plain TCP listener
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to start listener on %s: %w", addr, err)
		}
		log.Printf("🌐 Plain TCP C2 Listener started on %s (Profile: %s)", addr, profile.Name)
	}

	// Verify listener was created successfully
	if listener == nil {
		return fmt.Errorf("failed to create listener - listener is nil")
	}

	// Create listener instance
	instanceCtx, instanceCancel := context.WithCancel(ls.ctx)
	instance := &listenerInstance{
		listener: listener,
		profile:  profile,
		active:   true,
		ctx:      instanceCtx,
		cancel:   instanceCancel,
	}

	// Store the instance
	ls.listeners[profile.ID] = instance

	// Start accepting connections in a goroutine
	go ls.acceptConnections(instance)

	return nil
}

// checkPortAvailability checks if a port is available for binding
func (ls *ListenerService) checkPortAvailability(host string, port int) error {
	// Try to bind to the port temporarily to check availability
	testListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		// Suggest alternative ports
		suggestedPorts := ls.findAvailablePorts(host, port)
		if len(suggestedPorts) > 0 {
			return fmt.Errorf("port %d is already in use. Suggested available ports: %v", port, suggestedPorts)
		}
		return fmt.Errorf("port %d is already in use or not available", port)
	}
	testListener.Close()
	return nil
}

// findAvailablePorts finds available ports near the requested port
func (ls *ListenerService) findAvailablePorts(host string, requestedPort int) []int {
	var availablePorts []int
	// Check ports in range [requestedPort-5, requestedPort+5]
	for offset := -5; offset <= 5; offset++ {
		if offset == 0 {
			continue // Skip the requested port itself
		}
		testPort := requestedPort + offset
		if testPort >= 1024 && testPort <= 65535 { // Valid port range
			if testListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, testPort)); err == nil {
				testListener.Close()
				availablePorts = append(availablePorts, testPort)
				if len(availablePorts) >= 3 { // Limit to 3 suggestions
					break
				}
			}
		}
	}
	return availablePorts
}

// StopListener stops a specific listener by profile ID
func (ls *ListenerService) StopListener(profileID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	instance, exists := ls.listeners[profileID]
	if !exists {
		return fmt.Errorf("listener for profile ID '%s' not found", profileID)
	}

	if !instance.active {
		return nil
	}

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in StopListener: %v", r)
		}
	}()

	ls.stopListenerInternal(profileID)

	// Safe access to profile name
	profileName := "unknown"
	if instance.profile != nil {
		profileName = instance.profile.Name
	}

	log.Printf("🛑 C2 Listener stopped (Profile: %s)", profileName)
	return nil
}

// stopListenerInternal stops a specific listener without locking (internal use)
func (ls *ListenerService) stopListenerInternal(profileID string) {
	instance, exists := ls.listeners[profileID]
	if !exists {
		return
	}

	if instance.listener != nil {
		instance.listener.Close()
		instance.listener = nil
	}
	instance.active = false
	instance.cancel()

	// Remove from map
	delete(ls.listeners, profileID)
}

// StopAllListeners stops all active listeners
func (ls *ListenerService) StopAllListeners() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	for profileID := range ls.listeners {
		ls.stopListenerInternal(profileID)
	}

	log.Printf("🛑 All C2 Listeners stopped")
	return nil
}

// acceptConnections handles incoming connections
func (ls *ListenerService) acceptConnections(instance *listenerInstance) {
	for {
		select {
		case <-instance.ctx.Done():
			return
		default:
			// Set a timeout for accepting connections
			if tcpListener, ok := instance.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			conn, err := instance.listener.Accept()
			if err != nil {
				// Check if it's a timeout error (expected)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// Check if listener was closed
				if instance.ctx.Err() != nil {
					return
				}
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			// Handle the connection in a goroutine
			go ls.handleConnection(conn, instance)
		}
	}
}

// handleConnection handles an individual client connection
func (ls *ListenerService) handleConnection(conn net.Conn, instance *listenerInstance) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()

	// Determine connection type
	connType := "TCP"
	if tlsConn, ok := conn.(*tls.Conn); ok {
		connType = fmt.Sprintf("TLS %s", tlsConn.ConnectionState().Version)
		// Log TLS details
		state := tlsConn.ConnectionState()
		log.Printf("🔒 New TLS connection from %s - Version: %s, Cipher: %s",
			remoteAddr, tlsVersionString(state.Version), tls.CipherSuiteName(state.CipherSuite))
	} else {
		log.Printf("🔌 New TCP connection from %s", remoteAddr)
	}

	// Send welcome message with connection details
	profileName := "unknown"
	if instance.profile != nil {
		profileName = instance.profile.Name
	}
	welcomeMsg := fmt.Sprintf("Welcome to MuliC2 - Profile: %s\n", profileName)
	welcomeMsg += fmt.Sprintf("Connection: %s\n", connType)
	welcomeMsg += fmt.Sprintf("Remote: %s\n", remoteAddr)
	welcomeMsg += "PS > "
	conn.Write([]byte(welcomeMsg))

	// Enhanced C2 command handling
	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Connection closed from %s: %v", remoteAddr, err)
			break
		}

		command := string(buffer[:n])
		command = trimCommand(command)

		if command == "" {
			continue
		}

		// Handle special commands
		switch command {
		case "exit", "quit":
			log.Printf("Client %s requested disconnect", remoteAddr)
			conn.Write([]byte("Disconnecting...\n"))
			return
		case "version":
			version := "MuliC2 v1.0.0"
			if tlsConn, ok := conn.(*tls.Conn); ok {
				state := tlsConn.ConnectionState()
				version += fmt.Sprintf(" | TLS %s | Cipher: %s",
					tlsVersionString(state.Version), tls.CipherSuiteName(state.CipherSuite))
			}
			conn.Write([]byte(version + "\nPS > "))
		case "status":
			profileName := "unknown"
			if instance.profile != nil {
				profileName = instance.profile.Name
			}
			status := fmt.Sprintf("Active: true | Profile: %s | Connection: %s",
				profileName, connType)
			conn.Write([]byte(status + "\nPS > "))
		default:
			// Echo back the received command (placeholder for actual command execution)
			response := fmt.Sprintf("Command received: %s\nPS > ", command)
			conn.Write([]byte(response))
		}
	}
}

// trimCommand removes common command terminators
func trimCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.TrimSuffix(cmd, "\n")
	cmd = strings.TrimSuffix(cmd, "\r")
	cmd = strings.TrimSuffix(cmd, "\r\n")
	return cmd
}

// tlsVersionString converts TLS version to readable string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "1.3"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS10:
		return "1.0"
	default:
		return fmt.Sprintf("Unknown(%d)", version)
	}
}

// GetStatus returns the current listener status for all profiles
func (ls *ListenerService) GetStatus() map[string]interface{} {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	status := map[string]interface{}{
		"total_listeners":  len(ls.listeners),
		"active_listeners": 0,
		"profiles":         make(map[string]interface{}),
	}

	for profileID, instance := range ls.listeners {
		if instance.active {
			status["active_listeners"] = status["active_listeners"].(int) + 1
		}

		profileStatus := map[string]interface{}{
			"active": instance.active,
		}

		if instance.active && instance.profile != nil {
			profileStatus["profile"] = instance.profile
			profileStatus["address"] = fmt.Sprintf("%s:%d", instance.profile.Host, instance.profile.Port)
		}

		status["profiles"].(map[string]interface{})[profileID] = profileStatus
	}

	return status
}

// IsActive returns whether any listener is currently active
func (ls *ListenerService) IsActive() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	for _, instance := range ls.listeners {
		if instance.active {
			return true
		}
	}
	return false
}

// IsProfileActive returns whether a specific profile is active
func (ls *ListenerService) IsProfileActive(profileID string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	instance, exists := ls.listeners[profileID]
	return exists && instance.active
}

// GetActiveProfiles returns all currently active profiles
func (ls *ListenerService) GetActiveProfiles() []*Profile {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var activeProfiles []*Profile
	for _, instance := range ls.listeners {
		if instance.active && instance.profile != nil {
			activeProfiles = append(activeProfiles, instance.profile)
		}
	}
	return activeProfiles
}

// Close shuts down all listeners
func (ls *ListenerService) Close() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.cancel()

	// Close all listeners
	for profileID := range ls.listeners {
		ls.stopListenerInternal(profileID)
	}

	return nil
}
