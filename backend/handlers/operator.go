package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

type OperatorHandler struct {
	db *sql.DB
}

func NewOperatorHandler(db *sql.DB) *OperatorHandler {
	return &OperatorHandler{db: db}
}

type AgentInfo struct {
	ID          int       `json:"id"`
	Hostname    string    `json:"hostname"`
	Username    string    `json:"username"`
	IP          string    `json:"ip"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	ProfileID   string    `json:"profileId"`
	Status      string    `json:"status"`
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	ProfileName string    `json:"profileName"`
	ProfilePort string    `json:"profilePort"`
}

func (h *OperatorHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT a.id, a.hostname, a.username, a.ip, a.os, a.arch, a.profile_id, a.status, a.first_seen, a.last_seen,
		       p.name as profile_name, p.port as profile_port
		FROM agents a
		LEFT JOIN profiles p ON a.profile_id = p.id
		ORDER BY a.last_seen DESC
	`)
	if err != nil {
		log.Printf("List agents error: %v", err)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []AgentInfo
	for rows.Next() {
		var agent AgentInfo
		var profileName, profilePort sql.NullString
		err := rows.Scan(
			&agent.ID, &agent.Hostname, &agent.Username, &agent.IP, &agent.OS,
			&agent.Arch, &agent.ProfileID, &agent.Status, &agent.FirstSeen, &agent.LastSeen,
			&profileName, &profilePort,
		)
		if err != nil {
			log.Printf("Scan agent error: %v", err)
			continue
		}

		// Add profile information if available
		if profileName.Valid {
			agent.ProfileName = profileName.String
		}
		if profilePort.Valid {
			agent.ProfilePort = profilePort.String
		}

		agents = append(agents, agent)
	}

	log.Printf("📊 Listed %d agents", len(agents))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
	})
}

type EnqueueTaskRequest struct {
	AgentID int    `json:"agentId"`
	Command string `json:"command"`
}

type EnqueueTaskResponse struct {
	TaskID int    `json:"taskId"`
	Status string `json:"status"`
}

func (h *OperatorHandler) EnqueueTask(w http.ResponseWriter, r *http.Request) {
	var req EnqueueTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate agent exists and is online
	var agentStatus, profileID string
	err := h.db.QueryRow(`
		SELECT status, profile_id 
		FROM agents 
		WHERE id = $1
	`, req.AgentID).Scan(&agentStatus, &profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Task enqueue failed: Agent %d not found", req.AgentID)
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		log.Printf("Task enqueue error checking agent: %v", err)
		http.Error(w, "Failed to validate agent", http.StatusInternalServerError)
		return
	}

	if agentStatus != "online" {
		log.Printf("Task enqueue failed: Agent %d is not online (status: %s)", req.AgentID, agentStatus)
		http.Error(w, "Agent is not online", http.StatusBadRequest)
		return
	}

	var taskID int
	err = h.db.QueryRow(`
		INSERT INTO tasks (agent_id, command, status, created_at) 
		VALUES ($1, $2, 'pending', $3) 
		RETURNING id
	`, req.AgentID, req.Command, time.Now()).Scan(&taskID)
	if err != nil {
		log.Printf("Task enqueue error: %v", err)
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	log.Printf("📋 Task enqueued: ID=%d, Agent=%d, Profile=%s, Command='%s'",
		taskID, req.AgentID, profileID, req.Command)

	resp := EnqueueTaskResponse{TaskID: taskID, Status: "queued"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type TaskInfo struct {
	ID          int        `json:"id"`
	AgentID     int        `json:"agentId"`
	Command     string     `json:"command"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Output      *string    `json:"output,omitempty"`
}

func (h *OperatorHandler) GetAgentTasks(w http.ResponseWriter, r *http.Request) {
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

	rows, err := h.db.Query(`
		SELECT t.id, t.agent_id, t.command, t.status, t.created_at, t.started_at, t.completed_at, r.output
		FROM tasks t
		LEFT JOIN results r ON t.id = r.task_id
		WHERE t.agent_id = $1
		ORDER BY t.created_at DESC
	`, agentID)
	if err != nil {
		log.Printf("Get agent tasks error: %v", err)
		http.Error(w, "Failed to get agent tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []TaskInfo
	for rows.Next() {
		var task TaskInfo
		var startedAt, completedAt sql.NullTime
		var output sql.NullString

		err := rows.Scan(
			&task.ID, &task.AgentID, &task.Command, &task.Status,
			&task.CreatedAt, &startedAt, &completedAt, &output,
		)
		if err != nil {
			log.Printf("Scan task error: %v", err)
			continue
		}

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		if output.Valid {
			task.Output = &output.String
		}

		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
	})
}
