package services

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
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

// Validate validates the profile configuration
func (p *Profile) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("profile ID cannot be empty")
	}
	if p.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if p.Host == "" {
		p.Host = "0.0.0.0" // Default to all interfaces
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", p.Port)
	}
	if p.UseTLS {
		if p.CertFile == "" || p.KeyFile == "" {
			return fmt.Errorf("TLS is enabled but certificate files are not specified")
		}
		if _, err := os.Stat(p.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file not found: %s", p.CertFile)
		}
		if _, err := os.Stat(p.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("private key file not found: %s", p.KeyFile)
		}
	}
	return nil
}

// ListenerService manages C2 server listeners
type ListenerService struct {
	mu         sync.RWMutex
	listeners  map[string]*listenerInstance
	ctx        context.Context
	cancel     context.CancelFunc
	router     http.Handler
	vncService *VNCService
	wg         sync.WaitGroup // Track goroutines
}

// listenerInstance represents a single listener instance
type listenerInstance struct {
	listener    net.Listener
	profile     *Profile
	active      bool
	ctx         context.Context
	cancel      context.CancelFunc
	connections sync.Map // Track active connections
	connCount   int64    // Connection counter
}

// VNCService interface defines the contract for VNC service implementations
type VNCServiceInterface interface {
	HandleVNCConnection(conn net.Conn, remoteAddr string)
	Close() error
}

// VNCService handles VNC connections - implement this according to your needs
type VNCService struct {
	// Add your VNC service fields here
	active bool
	mu     sync.RWMutex
}

// NewVNCService creates a new VNC service instance
func NewVNCService() *VNCService {
	return &VNCService{
		active: true,
	}
}

// HandleVNCConnection handles incoming VNC connections
func (v *VNCService) HandleVNCConnection(conn net.Conn, remoteAddr string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active {
		conn.Close()
		return
	}

	defer conn.Close()

	log.Printf("🔍 VNC connection from %s - handling...", remoteAddr)

	// TODO: Implement your VNC handling logic here
	// For now, we'll just log and close the connection
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("VNC connection error from %s: %v", remoteAddr, err)
			}
			break
		}

		log.Printf("🔍 VNC data from %s: %d bytes", remoteAddr, n)
		// Process VNC data here
	}

	log.Printf("🔍 VNC connection from %s closed", remoteAddr)
}

// Close shuts down the VNC service
func (v *VNCService) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.active = false
	log.Printf("🔍 VNC service closed")
	return nil
}
func NewListenerService() *ListenerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ListenerService{
		listeners:  make(map[string]*listenerInstance),
		ctx:        ctx,
		cancel:     cancel,
		vncService: NewVNCService(),
	}
}

// SetRouter sets the HTTP router for unified API mode
func (ls *ListenerService) SetRouter(router http.Handler) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.router = router
}

// GetVNCService returns the VNC service instance
func (ls *ListenerService) GetVNCService() *VNCService {
	return ls.vncService
}

// createTLSConfig creates TLS configuration with enhanced compatibility
func (ls *ListenerService) createTLSConfig(profile *Profile) (*tls.Config, error) {
	// Load certificates
	cert, err := tls.LoadX509KeyPair(profile.CertFile, profile.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate files: %w", err)
	}
	log.Printf("🔒 Loaded certificate from %s and %s", profile.CertFile, profile.KeyFile)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,

		// Enhanced cipher suites for better compatibility
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,

			// TLS 1.2 cipher suites for PowerShell/.NET compatibility
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

			// Additional RSA cipher suites for broader compatibility
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		},

		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		ClientAuth:             tls.NoClientCert,
		SessionTicketsDisabled: false,

		// Enhanced logging
		GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			log.Printf("🔒 TLS ClientHello: SNI=%s", clientHello.ServerName)
			return &cert, nil
		},
	}

	log.Printf("🔒 TLS Configuration: Min=%s, Max=%s, Ciphers=%d",
		tlsVersionToString(tlsConfig.MinVersion),
		tlsVersionToString(tlsConfig.MaxVersion),
		len(tlsConfig.CipherSuites))

	return tlsConfig, nil
}

// tlsVersionToString converts TLS version constants to readable strings
func tlsVersionToString(version uint16) string {
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
		return fmt.Sprintf("Unknown(0x%04x)", version)
	}
}

// StartListener starts a new C2 listener with the specified profile
func (ls *ListenerService) StartListener(profile *Profile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}

	// Validate profile
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Generate unique profile ID if conflicts exist
	uniqueID := ls.generateUniqueID(profile.ID)
	if uniqueID != profile.ID {
		log.Printf("⚠️ Profile ID conflict. Generated unique ID: %s -> %s", profile.ID, uniqueID)
		profile.ID = uniqueID
	}

	// Check port availability
	if err := ls.checkPortAvailability(profile.Host, profile.Port); err != nil {
		return fmt.Errorf("port check failed: %w", err)
	}

	// Warn about privileged ports
	if profile.Port < 1024 {
		log.Printf("⚠️ WARNING: Port %d is privileged. Ensure proper permissions.", profile.Port)
	}

	// Create listener
	listener, err := ls.createListener(profile)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Create instance
	instanceCtx, instanceCancel := context.WithCancel(ls.ctx)
	instance := &listenerInstance{
		listener: listener,
		profile:  profile,
		active:   true,
		ctx:      instanceCtx,
		cancel:   instanceCancel,
	}

	ls.listeners[profile.ID] = instance

	// Start accepting connections
	ls.wg.Add(1)
	go ls.acceptConnections(instance)

	return nil
}

// generateUniqueID generates a unique profile ID
func (ls *ListenerService) generateUniqueID(baseID string) string {
	if _, exists := ls.listeners[baseID]; !exists {
		return baseID
	}

	for i := 1; i <= 1000; i++ {
		uniqueID := fmt.Sprintf("%s_%d", baseID, i)
		if _, exists := ls.listeners[uniqueID]; !exists {
			return uniqueID
		}
	}

	// Fallback with timestamp
	return fmt.Sprintf("%s_%d", baseID, time.Now().Unix())
}

// createListener creates the appropriate listener type
func (ls *ListenerService) createListener(profile *Profile) (net.Listener, error) {
	addr := fmt.Sprintf("%s:%d", profile.Host, profile.Port)

	if profile.UseTLS {
		tlsConfig, err := ls.createTLSConfig(profile)
		if err != nil {
			return nil, fmt.Errorf("TLS config creation failed: %w", err)
		}

		tcpListener, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("TCP listener creation failed: %w", err)
		}

		tlsListener := tls.NewListener(tcpListener, tlsConfig)
		log.Printf("🔒 TLS C2 Listener started on %s (Profile: %s)", addr, profile.Name)
		return tlsListener, nil
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("TCP listener creation failed: %w", err)
	}

	log.Printf("🌐 TCP C2 Listener started on %s (Profile: %s)", addr, profile.Name)
	return listener, nil
}

// checkPortAvailability checks if a port is available
func (ls *ListenerService) checkPortAvailability(host string, port int) error {
	testListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		suggestedPorts := ls.findAvailablePorts(host, port)
		if len(suggestedPorts) > 0 {
			return fmt.Errorf("port %d unavailable. Suggested ports: %v", port, suggestedPorts)
		}
		return fmt.Errorf("port %d unavailable", port)
	}
	testListener.Close()
	return nil
}

// findAvailablePorts finds available ports near the requested port
func (ls *ListenerService) findAvailablePorts(host string, requestedPort int) []int {
	var availablePorts []int

	for offset := -5; offset <= 5 && len(availablePorts) < 3; offset++ {
		if offset == 0 {
			continue
		}

		testPort := requestedPort + offset
		if testPort < 1024 || testPort > 65535 {
			continue
		}

		if testListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, testPort)); err == nil {
			testListener.Close()
			availablePorts = append(availablePorts, testPort)
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
		return nil // Already stopped
	}

	ls.stopListenerInternal(profileID)

	profileName := "unknown"
	if instance.profile != nil {
		profileName = instance.profile.Name
	}

	log.Printf("🛑 C2 Listener stopped (Profile: %s)", profileName)
	return nil
}

// stopListenerInternal stops a listener without locking
func (ls *ListenerService) stopListenerInternal(profileID string) {
	instance, exists := ls.listeners[profileID]
	if !exists || !instance.active {
		return
	}

	// Close all connections
	instance.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		instance.connections.Delete(key)
		return true
	})

	// Close listener
	if instance.listener != nil {
		instance.listener.Close()
		instance.listener = nil
	}

	instance.active = false
	instance.cancel()
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

// acceptConnections handles incoming connections with proper error handling
func (ls *ListenerService) acceptConnections(instance *listenerInstance) {
	defer ls.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in acceptConnections: %v", r)
		}
	}()

	for {
		select {
		case <-instance.ctx.Done():
			return
		default:
			// Set timeout for non-blocking accept
			if tcpListener, ok := instance.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			conn, err := instance.listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Expected timeout
				}
				if instance.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("Accept error: %v", err)
				continue
			}

			// Track connection
			connID := fmt.Sprintf("conn_%d_%d", instance.connCount, time.Now().UnixNano())
			instance.connCount++
			instance.connections.Store(connID, conn)

			// Handle connection
			ls.wg.Add(1)
			go func() {
				defer ls.wg.Done()
				defer func() {
					instance.connections.Delete(connID)
				}()
				ls.handleConnection(conn, instance, connID)
			}()
		}
	}
}

// handleConnection handles an individual client connection
func (ls *ListenerService) handleConnection(conn net.Conn, instance *listenerInstance, connID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleConnection: %v", r)
		}
		conn.Close()
	}()

	remoteAddr := conn.RemoteAddr().String()
	connType := "TCP"

	// Enhanced TLS logging
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// Set handshake timeout
		conn.SetDeadline(time.Now().Add(30 * time.Second))

		// Perform handshake
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("TLS handshake failed from %s: %v", remoteAddr, err)
			return
		}

		// Remove deadline after successful handshake
		conn.SetDeadline(time.Time{})

		state := tlsConn.ConnectionState()
		tlsVersion := tlsVersionToString(state.Version)
		connType = fmt.Sprintf("TLS %s", tlsVersion)

		log.Printf("🔒 TLS connection from %s: %s, Cipher: %s",
			remoteAddr, tlsVersion, tls.CipherSuiteName(state.CipherSuite))
	} else {
		log.Printf("🔌 TCP connection from %s", remoteAddr)
	}

	// VNC detection with better buffering
	vncDetected, bufferedConn := ls.detectVNCConnection(conn)
	if vncDetected {
		log.Printf("🔍 VNC connection detected from %s", remoteAddr)
		ls.vncService.HandleVNCConnection(bufferedConn, remoteAddr)
		return
	}

	// HTTP detection for unified API mode
	if ls.router != nil && ls.detectHTTPConnection(bufferedConn) {
		log.Printf("🌐 HTTP request detected from %s", remoteAddr)
		ls.handleHTTPRequest(&httpConn{Conn: bufferedConn, reader: bufio.NewReader(bufferedConn)})
		return
	}

	// Standard C2 handling
	ls.handleC2Connection(bufferedConn, instance, connType, remoteAddr)
}

// detectHTTPConnection detects if the connection is HTTP
func (ls *ListenerService) detectHTTPConnection(conn net.Conn) bool {
	reader := bufio.NewReader(conn)
	peek, err := reader.Peek(8)
	if err != nil {
		return false
	}

	peekStr := string(peek)
	httpMethods := []string{"GET ", "POST", "PUT ", "DELETE", "HEAD", "OPTIONS", "PATCH"}

	for _, method := range httpMethods {
		if strings.HasPrefix(peekStr, method) {
			return true
		}
	}

	return false
}

// detectVNCConnection detects VNC connections with enhanced PowerShell support
func (ls *ListenerService) detectVNCConnection(conn net.Conn) (bool, net.Conn) {
	reader := bufio.NewReader(conn)

	// Set a reasonable timeout for detection
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetReadDeadline(time.Time{}) // Clear deadline

	peekBytes, err := reader.Peek(8)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return false, conn
	}

	if len(peekBytes) >= 4 {
		// Check for VNC frame length (both endianness)
		frameLengthLE := binary.LittleEndian.Uint32(peekBytes[:4])
		frameLengthBE := binary.BigEndian.Uint32(peekBytes[:4])

		// PowerShell VNC detection (typical JPEG frames: 1KB-50KB)
		if (frameLengthLE >= 1024 && frameLengthLE <= 1024*50) ||
			(frameLengthBE >= 1024 && frameLengthBE <= 1024*50) {
			return true, &bufferedConn{Conn: conn, reader: reader}
		}

		// General VNC detection with broader range
		if (frameLengthLE >= 100 && frameLengthLE <= 1024*1024) ||
			(frameLengthBE >= 100 && frameLengthBE <= 1024*1024) {
			return true, &bufferedConn{Conn: conn, reader: reader}
		}

		// Check for RFB header
		if len(peekBytes) >= 3 && peekBytes[0] == 0x52 && peekBytes[1] == 0x46 && peekBytes[2] == 0x42 {
			return true, &bufferedConn{Conn: conn, reader: reader}
		}
	}

	return false, conn
}

// handleC2Connection handles standard C2 protocol
func (ls *ListenerService) handleC2Connection(conn net.Conn, instance *listenerInstance, connType, remoteAddr string) {
	// Send welcome message
	profileName := "unknown"
	if instance.profile != nil {
		profileName = instance.profile.Name
	}

	welcomeMsg := fmt.Sprintf("Welcome to MuliC2 - Profile: %s\nConnection: %s\nRemote: %s\nPS > ",
		profileName, connType, remoteAddr)
	conn.Write([]byte(welcomeMsg))

	// Command handling loop
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute)) // 5 minute timeout

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Connection read error from %s: %v", remoteAddr, err)
			}
			break
		}

		command := strings.TrimSpace(string(buffer[:n]))
		if command == "" {
			continue
		}

		// Reset read deadline on activity
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// Handle commands
		response := ls.handleCommand(command, connType, instance)
		if response == "DISCONNECT" {
			break
		}

		conn.Write([]byte(response))
	}
}

// handleCommand processes C2 commands
func (ls *ListenerService) handleCommand(command, connType string, instance *listenerInstance) string {
	switch strings.ToLower(command) {
	case "exit", "quit":
		return "Disconnecting...\n"

	case "version":
		return fmt.Sprintf("MuliC2 v1.0.0 | Connection: %s\nPS > ", connType)

	case "status":
		profileName := "unknown"
		if instance.profile != nil {
			profileName = instance.profile.Name
		}
		return fmt.Sprintf("Active: %v | Profile: %s | Type: %s\nPS > ",
			instance.active, profileName, connType)

	case "help":
		help := "Available commands:\n"
		help += "  version - Show version info\n"
		help += "  status  - Show connection status\n"
		help += "  help    - Show this help\n"
		help += "  exit    - Disconnect\nPS > "
		return help

	default:
		return fmt.Sprintf("Command received: %s\nPS > ", command)
	}
}

// Connection wrapper types
type httpConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *httpConn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

// handleHTTPRequest handles HTTP requests in unified mode
func (ls *ListenerService) handleHTTPRequest(conn *httpConn) {
	defer conn.Close()

	responseWriter := &httpResponseWriter{
		conn:       conn,
		header:     make(http.Header),
		statusCode: 200,
	}

	req, err := http.ReadRequest(conn.reader)
	if err != nil {
		log.Printf("HTTP request read error: %v", err)
		return
	}
	defer req.Body.Close()

	ls.router.ServeHTTP(responseWriter, req)
}

// httpResponseWriter implements http.ResponseWriter
type httpResponseWriter struct {
	conn        *httpConn
	header      http.Header
	statusCode  int
	wroteHeader bool
}

func (w *httpResponseWriter) Header() http.Header {
	return w.header
}

func (w *httpResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(data)
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true

	statusText := http.StatusText(statusCode)
	if statusText == "" {
		statusText = "Unknown"
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText)
	w.conn.Write([]byte(statusLine))

	for key, values := range w.header {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			w.conn.Write([]byte(headerLine))
		}
	}

	w.conn.Write([]byte("\r\n"))
}

// Status and management methods
func (ls *ListenerService) GetStatus() map[string]interface{} {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	activeCount := 0
	profiles := make(map[string]interface{})

	for profileID, instance := range ls.listeners {
		if instance.active {
			activeCount++
		}

		profileStatus := map[string]interface{}{
			"active": instance.active,
		}

		if instance.active && instance.profile != nil {
			profileStatus["profile"] = instance.profile
			profileStatus["address"] = fmt.Sprintf("%s:%d", instance.profile.Host, instance.profile.Port)
			profileStatus["tls_enabled"] = instance.profile.UseTLS

			// Count active connections
			connectionCount := 0
			instance.connections.Range(func(_, _ interface{}) bool {
				connectionCount++
				return true
			})
			profileStatus["active_connections"] = connectionCount
		}

		profiles[profileID] = profileStatus
	}

	return map[string]interface{}{
		"total_listeners":  len(ls.listeners),
		"active_listeners": activeCount,
		"profiles":         profiles,
	}
}

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

func (ls *ListenerService) IsProfileActive(profileID string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	instance, exists := ls.listeners[profileID]
	return exists && instance.active
}

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

// Close shuts down all listeners and waits for goroutines to finish
func (ls *ListenerService) Close() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Cancel context to signal all goroutines to stop
	ls.cancel()

	// Stop all listeners
	for profileID := range ls.listeners {
		ls.stopListenerInternal(profileID)
	}

	// Close VNC service
	if ls.vncService != nil {
		ls.vncService.Close()
	}

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		ls.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("🛑 ListenerService closed gracefully")
	case <-time.After(10 * time.Second):
		log.Printf("🛑 ListenerService close timeout - some goroutines may still be running")
	}

	return nil
}
