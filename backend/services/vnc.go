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
}

// NewVNCService creates a new VNC service
func NewVNCService() *VNCService {
	return &VNCService{
		connections:  make(map[string]*VNCConnection),
		frameChannel: make(chan VNCFrame, 1000), // Large buffer to prevent blocking
	}
}

// HandleVNCConnection processes a new VNC connection
func (vs *VNCService) HandleVNCConnection(conn net.Conn, agentIP string) {
	connectionID := fmt.Sprintf("vnc_%s_%d", agentIP, time.Now().Unix())

	vncConn := &VNCConnection{
		ID:          connectionID,
		AgentIP:     agentIP,
		Hostname:    "Unknown",
		Resolution:  "200x150",
		FPS:         5,
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

// processVNCStream processes the incoming VNC stream with maximum stability
func (vs *VNCService) processVNCStream(vncConn *VNCConnection) {
	defer func() {
		// Cleanup when connection closes
		vs.mu.Lock()
		delete(vs.connections, vncConn.ID)
		vs.mu.Unlock()

		// Don't close the connection here - let the client close it
		log.Printf("🔍 VNC connection processing ended: %s", vncConn.ID)
	}()

	// Set a very long read timeout to prevent premature closure
	vncConn.conn.SetReadDeadline(time.Now().Add(24 * time.Hour)) // 24 hour timeout

	log.Printf("🔍 DEBUG: Starting VNC stream processing for %s", vncConn.ID)

	for vncConn.IsActive {
		// Read frame length (4 bytes)
		lengthBytes := make([]byte, 4)
		n, err := io.ReadFull(vncConn.conn, lengthBytes)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				log.Printf("🔍 VNC client closed connection: %s", vncConn.ID)
			} else {
				log.Printf("🔍 Error reading frame length from %s: %v", vncConn.ID, err)
			}
			break
		}

		// Parse frame length (PowerShell sends little-endian)
		frameLength := binary.LittleEndian.Uint32(lengthBytes)
		log.Printf("🔍 DEBUG: Received frame length: %d bytes", frameLength)

		// Validate frame length
		if frameLength < 100 || frameLength > 1024*100 {
			log.Printf("🔍 Invalid frame length: %d bytes, skipping", frameLength)
			continue
		}

		// Check for termination signal
		if frameLength == 9 {
			terminationBytes := make([]byte, 9)
			_, err := io.ReadFull(vncConn.conn, terminationBytes)
			if err == nil && string(terminationBytes) == "TERMINATE" {
				log.Printf("🔍 VNC agent requested termination: %s", vncConn.ID)
				break
			}
		}

		// Read frame data
		frameData := make([]byte, frameLength)
		n, err = io.ReadFull(vncConn.conn, frameData)
		if err != nil {
			log.Printf("🔍 Error reading frame data: %v", err)
			break
		}

		log.Printf("🔍 DEBUG: Successfully read frame data: %d bytes", n)

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

		// Send frame to frontend (non-blocking with large buffer)
		select {
		case vs.frameChannel <- frame:
			log.Printf("🔍 Frame #%d sent to frontend from %s (Size: %d bytes)",
				vncConn.FrameCount, vncConn.ID, frame.Size)
		default:
			log.Printf("🔍 Frame buffer full, dropping frame from %s", vncConn.ID)
		}

		log.Printf("🔍 DEBUG: Frame #%d processed successfully, waiting for next frame...", vncConn.FrameCount)

		// Reset read timeout to prevent connection closure
		vncConn.conn.SetReadDeadline(time.Now().Add(24 * time.Hour))
		log.Printf("🔍 DEBUG: Read timeout reset to 24 hours")
	}

	log.Printf("🔍 DEBUG: VNC stream processing loop ended for %s", vncConn.ID)
}

// GetFrameChannel returns the channel for receiving frames
func (vs *VNCService) GetFrameChannel() <-chan VNCFrame {
	return vs.frameChannel
}

// GetActiveConnections returns all active VNC connections
func (vs *VNCService) GetActiveConnections() []map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

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
