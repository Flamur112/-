package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AgentHandler struct {
	db *sql.DB
}

func NewAgentHandler(db *sql.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

type RegisterRequest struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	ProfileID string `json:"profileId"`
}

type RegisterResponse struct {
	AgentID         int    `json:"agentId"`
	PollingInterval int    `json:"pollingInterval"`
	Status          string `json:"status"`
}

func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Auto-detect IP address if not provided
	if req.IP == "" {
		req.IP = getClientIP(r)
		log.Printf("Auto-detected IP for agent: %s", req.IP)
	}

	// Validate profile exists and get polling interval
	var pollInterval int
	err := h.db.QueryRow(`SELECT poll_interval FROM profiles WHERE id = $1 AND is_active = true`, req.ProfileID).Scan(&pollInterval)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Agent registration failed: Profile '%s' not found or inactive", req.ProfileID)
			http.Error(w, "Invalid profile ID", http.StatusBadRequest)
			return
		}
		log.Printf("Agent registration error checking profile: %v", err)
		http.Error(w, "Failed to validate profile", http.StatusInternalServerError)
		return
	}

	// Use default polling interval if not set
	if pollInterval <= 0 {
		pollInterval = 5
	}

	var agentID int
	err = h.db.QueryRow(
		`INSERT INTO agents (hostname, username, ip, os, arch, profile_id, status, first_seen, last_seen) 
		 VALUES ($1, $2, $3, $4, $5, $6, 'online', $7, $7) RETURNING id`,
		req.Hostname, req.Username, req.IP, req.OS, req.Arch, req.ProfileID, time.Now(),
	).Scan(&agentID)
	if err != nil {
		log.Printf("Agent register error: %v", err)
		http.Error(w, "Failed to register agent", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Agent registered: ID=%d, Hostname=%s, IP=%s, Profile=%s, PollInterval=%d",
		agentID, req.Hostname, req.IP, req.ProfileID, pollInterval)

	resp := RegisterResponse{
		AgentID:         agentID,
		PollingInterval: pollInterval,
		Status:          "registered",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getClientIP extracts the real client IP address from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if commaIndex := strings.Index(ip, ","); commaIndex != -1 {
			return strings.TrimSpace(ip[:commaIndex])
		}
		return strings.TrimSpace(ip)
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	if ip := r.Header.Get("X-Client-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Fall back to remote address
	if r.RemoteAddr != "" {
		// Remove port if present
		if colonIndex := strings.LastIndex(r.RemoteAddr, ":"); colonIndex != -1 {
			return r.RemoteAddr[:colonIndex]
		}
		return r.RemoteAddr
	}

	return "unknown"
}

type HeartbeatRequest struct {
	AgentID int    `json:"agentId"`
	Status  string `json:"status"`
}

func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`UPDATE agents SET last_seen=$1, status=$2 WHERE id=$3`, time.Now(), req.Status, req.AgentID)
	if err != nil {
		http.Error(w, "Failed to update heartbeat", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type TaskDTO struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
}

func (h *AgentHandler) FetchTasks(w http.ResponseWriter, r *http.Request) {
	agentIDStr := r.URL.Query().Get("agentId")
	if agentIDStr == "" {
		http.Error(w, "agentId required", http.StatusBadRequest)
		return
	}
	agentID, err := strconv.Atoi(agentIDStr)
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}

	// Start transaction to fetch tasks and update status atomically
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction for task fetch: %v", err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Fetch pending tasks for this agent (profile-based routing is implicit through agent_id)
	rows, err := tx.Query(`
		SELECT id, command 
		FROM tasks 
		WHERE agent_id = $1 AND status = 'pending' 
		ORDER BY created_at ASC
	`, agentID)
	if err != nil {
		log.Printf("Failed to fetch tasks for agent %d: %v", agentID, err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []TaskDTO
	var taskIDs []int
	for rows.Next() {
		var t TaskDTO
		if err := rows.Scan(&t.ID, &t.Command); err != nil {
			log.Printf("Failed to scan task: %v", err)
			continue
		}
		tasks = append(tasks, t)
		taskIDs = append(taskIDs, t.ID)
	}

	// Update task status to 'running' for fetched tasks
	if len(taskIDs) > 0 {
		placeholders := make([]string, len(taskIDs))
		args := make([]interface{}, len(taskIDs)+2)
		args[0] = "running"
		args[1] = time.Now()

		for i, id := range taskIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+3)
			args[i+2] = id
		}

		query := fmt.Sprintf(`
			UPDATE tasks 
			SET status = $1, started_at = $2
			WHERE id IN (%s)
		`, strings.Join(placeholders, ","))

		_, err = tx.Exec(query, args...)
		if err != nil {
			log.Printf("Failed to update task status: %v", err)
			http.Error(w, "Failed to update task status", http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit task fetch transaction: %v", err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	log.Printf("📋 Agent %d fetched %d tasks", agentID, len(tasks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

type ResultRequest struct {
	TaskID int    `json:"taskId"`
	Output string `json:"output"`
}

func (h *AgentHandler) SubmitResult(w http.ResponseWriter, r *http.Request) {
	var req ResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate task exists and belongs to a valid agent
	var agentID int
	var taskStatus string
	err := h.db.QueryRow(`
		SELECT t.agent_id, t.status 
		FROM tasks t 
		JOIN agents a ON t.agent_id = a.id 
		WHERE t.id = $1 AND a.status != 'offline'
	`, req.TaskID).Scan(&agentID, &taskStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Result submission failed: Task %d not found or agent offline", req.TaskID)
			http.Error(w, "Task not found or agent offline", http.StatusNotFound)
			return
		}
		log.Printf("Result submission error checking task: %v", err)
		http.Error(w, "Failed to validate task", http.StatusInternalServerError)
		return
	}

	// Validate task status
	if taskStatus != "running" && taskStatus != "pending" {
		log.Printf("Result submission failed: Task %d has invalid status '%s'", req.TaskID, taskStatus)
		http.Error(w, "Task has invalid status for result submission", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction for result submission: %v", err)
		http.Error(w, "Failed to save result", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Save result
	if _, err := tx.Exec(`
		INSERT INTO results (task_id, output, completed_at) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (task_id) DO UPDATE SET 
			output = $2, 
			completed_at = $3
	`, req.TaskID, req.Output, time.Now()); err != nil {
		log.Printf("Failed to save result for task %d: %v", req.TaskID, err)
		http.Error(w, "Failed to save result", http.StatusInternalServerError)
		return
	}

	// Update task status to completed
	if _, err := tx.Exec(`UPDATE tasks SET status = 'completed', completed_at = $1 WHERE id = $2`, time.Now(), req.TaskID); err != nil {
		log.Printf("Failed to update task status for task %d: %v", req.TaskID, err)
		http.Error(w, "Failed to update task status", http.StatusInternalServerError)
		return
	}

	// Update agent last_seen
	if _, err := tx.Exec(`UPDATE agents SET last_seen = $1 WHERE id = $2`, time.Now(), agentID); err != nil {
		log.Printf("Failed to update agent last_seen for agent %d: %v", agentID, err)
		// Don't fail the request for this, just log it
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit result submission transaction: %v", err)
		http.Error(w, "Failed to save result", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Result submitted: Task=%d, Agent=%d, OutputLength=%d",
		req.TaskID, agentID, len(req.Output))

	w.WriteHeader(http.StatusOK)
}

// ListAgents lists all agents with status
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT id, hostname, username, ip, os, arch, profile_id, status, first_seen, last_seen FROM agents ORDER BY last_seen DESC`)
	if err != nil {
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AgentDTO struct {
		ID        int       `json:"id"`
		Hostname  string    `json:"hostname"`
		Username  string    `json:"username"`
		IP        string    `json:"ip"`
		OS        string    `json:"os"`
		Arch      string    `json:"arch"`
		ProfileID string    `json:"profileId"`
		Status    string    `json:"status"`
		FirstSeen time.Time `json:"firstSeen"`
		LastSeen  time.Time `json:"lastSeen"`
	}

	var agents []AgentDTO
	for rows.Next() {
		var a AgentDTO
		if err := rows.Scan(&a.ID, &a.Hostname, &a.Username, &a.IP, &a.OS, &a.Arch, &a.ProfileID, &a.Status, &a.FirstSeen, &a.LastSeen); err != nil {
			http.Error(w, "Failed to parse agents", http.StatusInternalServerError)
			return
		}
		agents = append(agents, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"agents": agents})
}

// EnqueueTask creates a new task for an agent
func (h *AgentHandler) EnqueueTask(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		AgentID int    `json:"agentId"`
		Command string `json:"command"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AgentID == 0 || req.Command == "" {
		http.Error(w, "agentId and command required", http.StatusBadRequest)
		return
	}
	var taskID int
	if err := h.db.QueryRow(`INSERT INTO tasks (agent_id, command) VALUES ($1,$2) RETURNING id`, req.AgentID, req.Command).Scan(&taskID); err != nil {
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"taskId": taskID, "status": "queued"})
}

// ListTasks lists tasks for an agent
func (h *AgentHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	agentIDStr := r.URL.Query().Get("agentId")
	if agentIDStr == "" {
		http.Error(w, "agentId required", http.StatusBadRequest)
		return
	}
	agentID, err := strconv.Atoi(agentIDStr)
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}
	rows, err := h.db.Query(`SELECT id, command, status, created_at FROM tasks WHERE agent_id=$1 ORDER BY id DESC`, agentID)
	if err != nil {
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type TaskRow struct {
		ID        int       `json:"id"`
		Command   string    `json:"command"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var out []TaskRow
	for rows.Next() {
		var t TaskRow
		if err := rows.Scan(&t.ID, &t.Command, &t.Status, &t.CreatedAt); err != nil {
			http.Error(w, "Failed to parse tasks", http.StatusInternalServerError)
			return
		}
		out = append(out, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": out})
}
