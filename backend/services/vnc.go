package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// VNCConnection represents a VNC connection from an agent
type VNCConnection struct {
	ID          string
	AgentIP     string
	Hostname    string
	Resolution  string
	FPS         int
	ConnectedAt time.Time
	LastFrame   time.Time
	FrameCount  int
	IsActive    bool
	conn        net.Conn
	mu          sync.RWMutex
}

// VNCFrame represents a single VNC frame
type VNCFrame struct {
	ConnectionID string    `json:"connection_id"`
	Timestamp    time.Time `json:"timestamp"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Data         []byte    `json:"data"`
	Size         int       `json:"size"`
}

// VNCService manages VNC connections and frame processing
type VNCService struct {
	connections map[string]*VNCConnection
	mu          sync.RWMutex
	// Channel for broadcasting frames to frontend
	frameChannel chan VNCFrame
	// Add graceful shutdown support
	shutdown chan bool
}

// NewVNCService creates a new VNC service
func NewVNCService() *VNCService {
	return &VNCService{
		connections:  make(map[string]*VNCConnection),
		frameChannel: make(chan VNCFrame, 100), // Buffer 100 frames
		shutdown:     make(chan bool),
	}
}

// HandleVNCConnection processes a new VNC connection with better error handling
func (vs *VNCService) HandleVNCConnection(conn net.Conn, agentIP string) {
	connectionID := fmt.Sprintf("vnc_%s_%d", agentIP, time.Now().Unix())

	// Set connection timeouts and buffer sizes
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true)
		// Set socket buffer sizes
		tcpConn.SetReadBuffer(1024 * 64)  // 64KB read buffer
		tcpConn.SetWriteBuffer(1024 * 64) // 64KB write buffer
	}

	vncConn := &VNCConnection{
		ID:          connectionID,
		AgentIP:     agentIP,
		Hostname:    "Unknown", // Will be updated when we receive agent info
		Resolution:  "200x150", // Default from PowerShell script
		FPS:         5,         // Default from PowerShell script
		ConnectedAt: time.Now(),
		LastFrame:   time.Now(),
		FrameCount:  0,
		IsActive:    true,
		conn:        conn,
	}

	// Store the connection
	vs.mu.Lock()
	vs.connections[connectionID] = vncConn
	vs.mu.Unlock()

	log.Printf("🔍 New VNC connection established: %s from %s", connectionID, agentIP)
	log.Printf("🔍 Total VNC connections: %d", len(vs.connections))

	// Send acknowledgment to client to confirm connection is ready
	vs.sendAcknowledgment(conn)

	// Start processing frames from this connection
	go vs.processVNCStream(vncConn)
}

// sendAcknowledgment sends a simple acknowledgment to the client
func (vs *VNCService) sendAcknowledgment(conn net.Conn) {
	// Send a simple "OK" response to let client know server is ready
	ackMsg := []byte("VNC_READY")
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Write(ackMsg)
	if err != nil {
		log.Printf("🔍 Warning: Could not send acknowledgment: %v", err)
	}
}

// processVNCStream processes the incoming VNC stream with robust error handling
func (vs *VNCService) processVNCStream(vncConn *VNCConnection) {
	defer func() {
		vs.mu.Lock()
		delete(vs.connections, vncConn.ID)
		vs.mu.Unlock()
		vncConn.conn.Close()
		log.Printf("🔍 VNC connection closed: %s", vncConn.ID)
	}()

	log.Printf("🔍 Starting VNC stream processing for %s", vncConn.ID)

	buffer := make([]byte, 8192)
	var pendingData []byte

	for vncConn.IsActive {
		select {
		case <-vs.shutdown:
			log.Printf("🔍 Shutdown signal received for %s", vncConn.ID)
			return
		default:
		}

		// Read more data from the connection
		vncConn.conn.SetReadDeadline(time.Now().Add(15 * time.Second)) // Longer timeout
		n, err := vncConn.conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("🔍 Read timeout for %s, continuing...", vncConn.ID)
				continue
			}
			if err == io.EOF {
				log.Printf("🔍 VNC client closed connection cleanly: %s", vncConn.ID)
				break
			}
			log.Printf("🔍 Error reading from %s: %v", vncConn.ID, err)
			break
		}
		if n == 0 {
			log.Printf("🔍 No data read from %s, connection may be closed", vncConn.ID)
			break
		}

		// Add new data to pending buffer
		pendingData = append(pendingData, buffer[:n]...)
		log.Printf("🔍 Read %d bytes from %s (total pending: %d)", n, vncConn.ID, len(pendingData))

		// Process all complete frames in the buffer
		for len(pendingData) >= 4 {
			frameLength := binary.LittleEndian.Uint32(pendingData[:4])
			totalFrameSize := 4 + int(frameLength)

			log.Printf("🔍 Frame length: %d, total frame size: %d, pending data: %d",
				frameLength, totalFrameSize, len(pendingData))

			// Validate frame length
			if frameLength < 10 || frameLength > 1024*200 {
				log.Printf("🔍 Invalid frame length from %s: %d bytes, resyncing buffer", vncConn.ID, frameLength)
				pendingData = pendingData[1:]
				continue
			}

			// Check for termination signal
			if frameLength == 9 && len(pendingData) >= 13 {
				terminationData := pendingData[4:13]
				if string(terminationData) == "TERMINATE" {
					log.Printf("🔍 VNC agent requested termination: %s", vncConn.ID)
					return
				}
			}

			// If we don't have the complete frame yet, wait for more data
			if len(pendingData) < totalFrameSize {
				log.Printf("🔍 Incomplete frame, waiting for more data (have %d, need %d)",
					len(pendingData), totalFrameSize)
				break // Exit inner loop, continue reading more data
			}

			// Extract complete frame data
			frameData := make([]byte, frameLength)
			copy(frameData, pendingData[4:4+frameLength])

			// Remove processed frame from pending data
			if len(pendingData) > totalFrameSize {
				pendingData = pendingData[totalFrameSize:]
			} else {
				pendingData = nil
			}

			// Process the frame
			vs.processFrame(vncConn, frameData)
		}
	}
}

// Alternative approach - More robust frame reading with dedicated function
func (vs *VNCService) readCompleteFrame(conn net.Conn, buffer []byte, pendingData *[]byte) ([]byte, error) {
	// Make sure we have at least the frame header (4 bytes)
	for len(*pendingData) < 4 {
		n, err := conn.Read(buffer)
		if err != nil {
			return nil, err
		}
		*pendingData = append(*pendingData, buffer[:n]...)
	}

	// Get frame length
	frameLength := binary.LittleEndian.Uint32((*pendingData)[:4])
	totalFrameSize := 4 + int(frameLength)

	// Validate frame length
	if frameLength < 10 || frameLength > 1024*200 {
		return nil, fmt.Errorf("invalid frame length: %d", frameLength)
	}

	// Read until we have the complete frame
	for len(*pendingData) < totalFrameSize {
		conn.SetReadDeadline(time.Now().Add(15 * time.Second)) // Longer timeout for large frames
		n, err := conn.Read(buffer)
		if err != nil {
			return nil, err
		}
		*pendingData = append(*pendingData, buffer[:n]...)
		log.Printf("🔍 Reading frame data: have %d/%d bytes", len(*pendingData), totalFrameSize)
	}

	// Extract the complete frame
	frameData := make([]byte, frameLength)
	copy(frameData, (*pendingData)[4:4+frameLength])

	// Remove processed frame from pending data
	if len(*pendingData) > totalFrameSize {
		*pendingData = (*pendingData)[totalFrameSize:]
	} else {
		*pendingData = nil
	}

	return frameData, nil
}

// processFrame handles individual frame processing
func (vs *VNCService) processFrame(vncConn *VNCConnection, frameData []byte) {
	// Update connection stats
	vncConn.mu.Lock()
	vncConn.FrameCount++
	vncConn.LastFrame = time.Now()
	vncConn.mu.Unlock()

	// Create VNC frame
	frame := VNCFrame{
		ConnectionID: vncConn.ID,
		Timestamp:    time.Now(),
		Width:        200,
		Height:       150,
		Data:         frameData,
		Size:         len(frameData),
	}

	// Send frame to frontend (non-blocking)
	select {
	case vs.frameChannel <- frame:
		log.Printf("🔍 Frame #%d processed from %s (Size: %d bytes)",
			vncConn.FrameCount, vncConn.ID, frame.Size)
	default:
		log.Printf("🔍 Frame buffer full, dropping frame from %s", vncConn.ID)
	}
}

// cleanupConnection handles connection cleanup
func (vs *VNCService) cleanupConnection(vncConn *VNCConnection) {
	vncConn.mu.Lock()
	vncConn.IsActive = false
	vncConn.mu.Unlock()

	// Close connection gracefully
	if vncConn.conn != nil {
		vncConn.conn.Close()
	}

	// Remove from active connections
	vs.mu.Lock()
	delete(vs.connections, vncConn.ID)
	vs.mu.Unlock()

	log.Printf("🔍 VNC connection cleaned up: %s (processed %d frames)",
		vncConn.ID, vncConn.FrameCount)
}

// GetFrameChannel returns the channel for receiving frames
func (vs *VNCService) GetFrameChannel() <-chan VNCFrame {
	return vs.frameChannel
}

// GetActiveConnections returns all active VNC connections
func (vs *VNCService) GetActiveConnections() []map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	log.Printf("🔍 GetActiveConnections called, total connections: %d", len(vs.connections))

	var connections []map[string]interface{}
	for _, conn := range vs.connections {
		conn.mu.RLock()
		connectionInfo := map[string]interface{}{
			"id":           conn.ID,
			"agent_ip":     conn.AgentIP,
			"hostname":     conn.Hostname,
			"resolution":   conn.Resolution,
			"fps":          conn.FPS,
			"connected_at": conn.ConnectedAt,
			"last_frame":   conn.LastFrame,
			"frame_count":  conn.FrameCount,
			"is_active":    conn.IsActive,
		}
		conn.mu.RUnlock()
		connections = append(connections, connectionInfo)
		log.Printf("🔍 Connection %s: %s (%s) - Active: %v", conn.ID, conn.Hostname, conn.AgentIP, conn.IsActive)
	}

	return connections
}

// CloseConnection closes a specific VNC connection
func (vs *VNCService) CloseConnection(connectionID string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	conn, exists := vs.connections[connectionID]
	if !exists {
		return fmt.Errorf("VNC connection %s not found", connectionID)
	}

	conn.mu.Lock()
	conn.IsActive = false
	conn.mu.Unlock()

	conn.conn.Close()
	delete(vs.connections, connectionID)

	log.Printf("🔍 VNC connection closed: %s", connectionID)
	return nil
}

// CloseAllConnections closes all VNC connections
func (vs *VNCService) CloseAllConnections() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	for _, conn := range vs.connections {
		conn.mu.Lock()
		conn.IsActive = false
		conn.mu.Unlock()
		conn.conn.Close()
	}

	vs.connections = make(map[string]*VNCConnection)
	close(vs.shutdown)
	log.Printf("🔍 All VNC connections closed")
}

// Shutdown gracefully shuts down the VNC service
func (vs *VNCService) Shutdown() {
	log.Printf("🔍 Shutting down VNC service...")
	vs.CloseAllConnections()
	close(vs.frameChannel)
}
