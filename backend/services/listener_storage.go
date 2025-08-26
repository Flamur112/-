package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ListenerStorage manages persistent storage of listener configurations
type ListenerStorage struct {
	db *sql.DB
}

// StoredListener represents a listener configuration stored in the database
type StoredListener struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ProjectName string    `json:"projectName"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Description string    `json:"description"`
	UseTLS      bool      `json:"useTLS"`
	CertFile    string    `json:"certFile"`
	KeyFile     string    `json:"keyFile"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NewListenerStorage creates a new listener storage service
func NewListenerStorage(db *sql.DB) *ListenerStorage {
	return &ListenerStorage{db: db}
}

// Initialize creates the listeners table if it doesn't exist
func (ls *ListenerStorage) Initialize() error {
	query := `
	CREATE TABLE IF NOT EXISTS listeners (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		project_name VARCHAR(255),
		host VARCHAR(255) NOT NULL DEFAULT '0.0.0.0',
		port INTEGER NOT NULL,
		description TEXT,
		use_tls BOOLEAN NOT NULL DEFAULT true,
		cert_file VARCHAR(500),
		key_file VARCHAR(500),
		is_active BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_listeners_active ON listeners(is_active);
	CREATE INDEX IF NOT EXISTS idx_listeners_port ON listeners(port);
	`

	_, err := ls.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create listeners table: %w", err)
	}

	log.Printf("✅ Listeners table initialized")
	return nil
}

// SaveListener stores a listener configuration in the database
func (ls *ListenerStorage) SaveListener(listener *StoredListener) error {
	query := `
	INSERT INTO listeners (id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (id) DO UPDATE SET
		name = EXCLUDED.name,
		project_name = EXCLUDED.project_name,
		host = EXCLUDED.host,
		port = EXCLUDED.port,
		description = EXCLUDED.description,
		use_tls = EXCLUDED.use_tls,
		cert_file = EXCLUDED.cert_file,
		key_file = EXCLUDED.key_file,
		is_active = EXCLUDED.is_active,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := ls.db.Exec(query,
		listener.ID,
		listener.Name,
		listener.ProjectName,
		listener.Host,
		listener.Port,
		listener.Description,
		listener.UseTLS,
		listener.CertFile,
		listener.KeyFile,
		listener.IsActive,
		listener.CreatedAt,
		listener.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save listener: %w", err)
	}

	log.Printf("💾 Listener '%s' saved to database", listener.Name)
	return nil
}

// GetListener retrieves a listener by ID
func (ls *ListenerStorage) GetListener(id string) (*StoredListener, error) {
	query := `SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at FROM listeners WHERE id = $1`

	var listener StoredListener
	err := ls.db.QueryRow(query, id).Scan(
		&listener.ID,
		&listener.Name,
		&listener.ProjectName,
		&listener.Host,
		&listener.Port,
		&listener.Description,
		&listener.UseTLS,
		&listener.CertFile,
		&listener.KeyFile,
		&listener.IsActive,
		&listener.CreatedAt,
		&listener.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("listener %s not found", id)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	return &listener, nil
}

// GetAllListeners retrieves all stored listeners
func (ls *ListenerStorage) GetAllListeners() ([]*StoredListener, error) {
	query := `SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at FROM listeners ORDER BY created_at DESC`

	rows, err := ls.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query listeners: %w", err)
	}
	defer rows.Close()

	var listeners []*StoredListener
	for rows.Next() {
		var listener StoredListener
		err := rows.Scan(
			&listener.ID,
			&listener.Name,
			&listener.ProjectName,
			&listener.Host,
			&listener.Port,
			&listener.Description,
			&listener.UseTLS,
			&listener.CertFile,
			&listener.KeyFile,
			&listener.IsActive,
			&listener.CreatedAt,
			&listener.UpdatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Error scanning listener row: %v", err)
			continue
		}
		listeners = append(listeners, &listener)
	}

	return listeners, nil
}

// GetActiveListeners retrieves only active listeners
func (ls *ListenerStorage) GetActiveListeners() ([]*StoredListener, error) {
	query := `SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at FROM listeners WHERE is_active = true ORDER BY created_at DESC`

	rows, err := ls.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active listeners: %w", err)
	}
	defer rows.Close()

	var listeners []*StoredListener
	for rows.Next() {
		var listener StoredListener
		err := rows.Scan(
			&listener.ID,
			&listener.Name,
			&listener.ProjectName,
			&listener.Host,
			&listener.Port,
			&listener.Description,
			&listener.UseTLS,
			&listener.CertFile,
			&listener.KeyFile,
			&listener.IsActive,
			&listener.CreatedAt,
			&listener.UpdatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Error scanning listener row: %v", err)
			continue
		}
		listeners = append(listeners, &listener)
	}

	return listeners, nil
}

// UpdateListenerStatus updates the active status of a listener
func (ls *ListenerStorage) UpdateListenerStatus(id string, isActive bool) error {
	query := `UPDATE listeners SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	result, err := ls.db.Exec(query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update listener status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("listener %s not found", id)
	}

	log.Printf("💾 Listener '%s' status updated to active=%v", id, isActive)
	return nil
}

// DeleteListener removes a listener from the database
func (ls *ListenerStorage) DeleteListener(id string) error {
	query := `DELETE FROM listeners WHERE id = $1`

	result, err := ls.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("listener %s not found", id)
	}

	log.Printf("🗑️  Listener '%s' deleted from database", id)
	return nil
}

// LoadDefaultListeners loads the default listeners from config.json into the database
func (ls *ListenerStorage) LoadDefaultListeners(defaultProfiles []Profile) error {
	for _, profile := range defaultProfiles {
		// Check if this profile already exists
		existing, err := ls.GetListener(profile.ID)
		if err == nil && existing != nil {
			// Profile exists, skip
			continue
		}

		// Create new stored listener
		storedListener := &StoredListener{
			ID:          profile.ID,
			Name:        profile.Name,
			ProjectName: profile.ProjectName,
			Host:        profile.Host,
			Port:        profile.Port,
			Description: profile.Description,
			UseTLS:      profile.UseTLS,
			CertFile:    profile.CertFile,
			KeyFile:     profile.KeyFile,
			IsActive:    false, // Default to inactive
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := ls.SaveListener(storedListener); err != nil {
			log.Printf("⚠️  Failed to save default listener '%s': %v", profile.Name, err)
		} else {
			log.Printf("💾 Default listener '%s' loaded into database", profile.Name)
		}
	}

	return nil
}
