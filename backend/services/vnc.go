package services

import (
	"encoding/binary"
	"fmt"
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
}

// NewVNCService creates a new VNC service
func NewVNCService() *VNCService {
	return &VNCService{
		connections:  make(map[string]*VNCConnection),
		frameChannel: make(chan VNCFrame, 100), // Buffer 100 frames
	}
}

// HandleVNCConnection processes a new VNC connection
func (vs *VNCService) HandleVNCConnection(conn net.Conn, agentIP string) {
	connectionID := fmt.Sprintf("vnc_%s_%d", agentIP, time.Now().Unix())

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

	// Start processing frames from this connection
	go vs.processVNCStream(vncConn)
}

// processVNCStream processes the incoming VNC stream
func (vs *VNCService) processVNCStream(vncConn *VNCConnection) {
	defer func() {
		// Cleanup when connection closes
		vs.mu.Lock()
		delete(vs.connections, vncConn.ID)
		vs.mu.Unlock()
		vncConn.conn.Close()
		log.Printf("🔍 VNC connection closed: %s", vncConn.ID)
	}()

	// Set read timeout
	vncConn.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for vncConn.IsActive {
		// Read frame length (4 bytes)
		lengthBytes := make([]byte, 4)
		_, err := vncConn.conn.Read(lengthBytes)
		if err != nil {
			log.Printf("🔍 Error reading frame length from %s: %v", vncConn.ID, err)
			break
		}

		// Parse frame length - PowerShell VNC sends little-endian
		frameLength := binary.LittleEndian.Uint32(lengthBytes)
		if frameLength > 1024*1024 { // Max 1MB frame
			log.Printf("🔍 Frame too large from %s: %d bytes", vncConn.ID, frameLength)
			continue
		}

		// Check for termination signal
		if frameLength == 9 { // "TERMINATE" is 9 bytes
			terminationBytes := make([]byte, 9)
			_, err := vncConn.conn.Read(terminationBytes)
			if err == nil && string(terminationBytes) == "TERMINATE" {
				log.Printf("🔍 VNC agent requested termination: %s", vncConn.ID)
				break
			}
		}

		// Read frame data
		frameData := make([]byte, frameLength)
		_, err = vncConn.conn.Read(frameData)
		if err != nil {
			log.Printf("🔍 Error reading frame data from %s: %v", vncConn.ID, err)
			break
		}

		// Update connection stats
		vncConn.mu.Lock()
		vncConn.FrameCount++
		vncConn.LastFrame = time.Now()
		vncConn.mu.Unlock()

		// Create VNC frame
		frame := VNCFrame{
			ConnectionID: vncConn.ID,
			Timestamp:    time.Now(),
			Width:        200, // Default from PowerShell script
			Height:       150, // Default from PowerShell script
			Data:         frameData,
			Size:         len(frameData),
		}

		// Send frame to frontend (non-blocking)
		select {
		case vs.frameChannel <- frame:
			log.Printf("🔍 Frame #%d sent to frontend from %s (Size: %d bytes)",
				vncConn.FrameCount, vncConn.ID, frame.Size)
		default:
			log.Printf("🔍 Frame buffer full, dropping frame from %s", vncConn.ID)
		}

		// Reset read timeout
		vncConn.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	}
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

	conn.IsActive = false
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
		conn.IsActive = false
		conn.conn.Close()
	}

	vs.connections = make(map[string]*VNCConnection)
	log.Printf("🔍 All VNC connections closed")
}
