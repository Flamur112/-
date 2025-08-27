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

	// DO NOT send acknowledgment - PowerShell script doesn't expect it
	// vs.sendAcknowledgment(conn)

	// Start processing frames from this connection
	go vs.processVNCStream(vncConn)
}

// processVNCStream processes the incoming VNC stream with robust error handling
func (vs *VNCService) processVNCStream(vncConn *VNCConnection) {
	defer func() {
		log.Printf("🔍 VNC stream processing ended for %s, cleaning up", vncConn.ID)
		vs.cleanupConnection(vncConn)
	}()

	log.Printf("🔍 Starting VNC stream processing for %s", vncConn.ID)

	buffer := make([]byte, 8192)
	var pendingData []byte

	for {
		// Check if connection is still active
		vncConn.mu.RLock()
		isActive := vncConn.IsActive
		vncConn.mu.RUnlock()

		if !isActive {
			log.Printf("🔍 Connection marked inactive, stopping stream: %s", vncConn.ID)
			break
		}

		// Check for shutdown signal (non-blocking)
		select {
		case <-vs.shutdown:
			log.Printf("🔍 Shutdown signal received for %s", vncConn.ID)
			return
		default:
			// Continue processing
		}

		// Read more data from the connection with appropriate timeout
		vncConn.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Longer timeout for continuous stream
		n, err := vncConn.conn.Read(buffer)
		if err != nil {
			// Enhanced error logging with error type information
			log.Printf("🔍 Error reading from %s: %T %v", vncConn.ID, err, err)

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("🔍 Read timeout for %s, connection may be idle", vncConn.ID)
				// For continuous streams, timeout might be normal - check connection health
				continue
			}
			if err == io.EOF {
				log.Printf("🔍 VNC client closed connection cleanly: %s", vncConn.ID)
				break
			}
			// For other errors (including "connection aborted"), exit gracefully
			log.Printf("🔍 Network error on %s, closing connection: %v", vncConn.ID, err)
			break
		}

		if n == 0 {
			log.Printf("🔍 No data read from %s, connection may be closing", vncConn.ID)
			// Don't immediately break on zero bytes - could be temporary
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Add new data to pending buffer
		pendingData = append(pendingData, buffer[:n]...)
		log.Printf("🔍 Read %d bytes from %s (total pending: %d)", n, vncConn.ID, len(pendingData))

		// Process all complete frames in the buffer
		processedFrames := 0
		for len(pendingData) >= 4 {
			frameLength := binary.LittleEndian.Uint32(pendingData[:4])
			totalFrameSize := 4 + int(frameLength)

			log.Printf("🔍 Processing frame - length: %d, total size: %d, buffer: %d",
				frameLength, totalFrameSize, len(pendingData))

			// Validate frame length - be more permissive for small test images
			if frameLength < 10 || frameLength > 2*1024*1024 { // Allow up to 2MB frames
				log.Printf("🔍 Invalid frame length from %s: %d bytes, resyncing buffer", vncConn.ID, frameLength)
				// Look for next potential frame header
				pendingData = vs.findNextFrameHeader(pendingData[1:])
				continue
			}

			// Check for termination signal
			if frameLength >= 9 && len(pendingData) >= int(frameLength)+4 {
				frameStart := 4
				frameEnd := frameStart + int(frameLength)
				if frameEnd <= len(pendingData) && frameLength >= 9 {
					checkData := pendingData[frameStart : frameStart+9]
					if string(checkData) == "TERMINATE" {
						log.Printf("🔍 VNC agent requested termination: %s", vncConn.ID)
						return // This will trigger the defer cleanup
					}
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

			// Process the frame - CRITICAL: This should NOT close the connection
			vs.processFrame(vncConn, frameData)
			processedFrames++

			// Update last activity time
			vncConn.mu.Lock()
			vncConn.LastFrame = time.Now()
			vncConn.mu.Unlock()
		}

		if processedFrames > 0 {
			log.Printf("🔍 Processed %d frames from %s, continuing to listen for more...",
				processedFrames, vncConn.ID)
		}

		// CRITICAL: Continue the loop to keep processing more frames
		// DO NOT break or return here - the connection should stay open for continuous streaming
	}

	// This point should only be reached on actual connection errors or shutdown
	log.Printf("🔍 VNC stream processing loop ended for %s", vncConn.ID)
}

// findNextFrameHeader attempts to find the next valid frame header in the buffer
func (vs *VNCService) findNextFrameHeader(data []byte) []byte {
	for i := 0; i < len(data)-4; i++ {
		frameLength := binary.LittleEndian.Uint32(data[i : i+4])
		if frameLength >= 10 && frameLength <= 2*1024*1024 {
			log.Printf("🔍 Found potential frame header at offset %d, length %d", i, frameLength)
			return data[i:]
		}
	}
	// No valid header found, return empty slice
	log.Printf("🔍 No valid frame header found in %d bytes", len(data))
	return []byte{}
}

// processFrame handles individual frame processing - DOES NOT CLOSE CONNECTION
func (vs *VNCService) processFrame(vncConn *VNCConnection, frameData []byte) {
	// Update connection stats
	vncConn.mu.Lock()
	vncConn.FrameCount++
	frameCount := vncConn.FrameCount
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
		log.Printf("🔍 Frame #%d processed from %s (Size: %d bytes) - KEEPING CONNECTION OPEN",
			frameCount, vncConn.ID, frame.Size)
	default:
		log.Printf("🔍 Frame buffer full, dropping frame from %s", vncConn.ID)
	}

	// CRITICAL: DO NOT CLOSE CONNECTION HERE
	// The connection must remain open for continuous frame streaming
	// Only cleanup should happen when the connection is actually lost or terminated
}

// cleanupConnection handles connection cleanup
func (vs *VNCService) cleanupConnection(vncConn *VNCConnection) {
	log.Printf("🔍 Starting cleanup for connection: %s", vncConn.ID)

	vncConn.mu.Lock()
	wasActive := vncConn.IsActive
	vncConn.IsActive = false
	frameCount := vncConn.FrameCount
	vncConn.mu.Unlock()

	// Close connection gracefully if it was active
	if wasActive && vncConn.conn != nil {
		log.Printf("🔍 Closing network connection for: %s", vncConn.ID)
		vncConn.conn.Close()
	}

	// Remove from active connections
	vs.mu.Lock()
	delete(vs.connections, vncConn.ID)
	connectionCount := len(vs.connections)
	vs.mu.Unlock()

	log.Printf("🔍 VNC connection cleaned up: %s (processed %d frames), remaining connections: %d",
		vncConn.ID, frameCount, connectionCount)
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
		log.Printf("🔍 Connection %s: %s (%s) - Active: %v, Frames: %d",
			conn.ID, conn.Hostname, conn.AgentIP, conn.IsActive, conn.FrameCount)
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

	// Send shutdown signal to all stream processors
	select {
	case vs.shutdown <- true:
	default:
		// Channel might be full or closed
	}

	log.Printf("🔍 All VNC connections closed")
}

// Shutdown gracefully shuts down the VNC service
func (vs *VNCService) Shutdown() {
	log.Printf("🔍 Shutting down VNC service...")
	vs.CloseAllConnections()
	close(vs.frameChannel)
}
