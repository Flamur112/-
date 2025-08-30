package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ProfileStorage manages persistent storage of C2 profile configurations
type ProfileStorage struct {
	db *sql.DB
}

// StoredProfile represents a C2 profile configuration stored in the database
type StoredProfile struct {
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

// NewProfileStorage creates a new profile storage service
func NewProfileStorage(db *sql.DB) *ProfileStorage {
	return &ProfileStorage{db: db}
}

// SaveProfile stores a profile configuration in the database
func (ps *ProfileStorage) SaveProfile(profile *StoredProfile) error {
	query := `
	INSERT INTO profiles (id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at)
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

	_, err := ps.db.Exec(query,
		profile.ID,
		profile.Name,
		profile.ProjectName,
		profile.Host,
		profile.Port,
		profile.Description,
		profile.UseTLS,
		profile.CertFile,
		profile.KeyFile,
		profile.IsActive,
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	log.Printf("✅ Profile saved to database: %s", profile.ID)
	return nil
}

// GetAllProfiles retrieves all profiles from the database
func (ps *ProfileStorage) GetAllProfiles() ([]*StoredProfile, error) {
	query := `
	SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at
	FROM profiles
	ORDER BY created_at DESC
	`

	rows, err := ps.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*StoredProfile
	for rows.Next() {
		profile := &StoredProfile{}
		err := rows.Scan(
			&profile.ID,
			&profile.Name,
			&profile.ProjectName,
			&profile.Host,
			&profile.Port,
			&profile.Description,
			&profile.UseTLS,
			&profile.CertFile,
			&profile.KeyFile,
			&profile.IsActive,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, profile)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profiles: %w", err)
	}

	log.Printf("✅ Retrieved %d profiles from database", len(profiles))
	return profiles, nil
}

// GetProfile retrieves a specific profile by ID
func (ps *ProfileStorage) GetProfile(id string) (*StoredProfile, error) {
	query := `
	SELECT id, name, project_name, host, port, description, use_tls, cert_file, key_file, is_active, created_at, updated_at
	FROM profiles
	WHERE id = $1
	`

	profile := &StoredProfile{}
	err := ps.db.QueryRow(query, id).Scan(
		&profile.ID,
		&profile.Name,
		&profile.ProjectName,
		&profile.Host,
		&profile.Port,
		&profile.Description,
		&profile.UseTLS,
		&profile.CertFile,
		&profile.KeyFile,
		&profile.IsActive,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// UpdateProfileStatus updates the active status of a profile
func (ps *ProfileStorage) UpdateProfileStatus(id string, isActive bool) error {
	query := `
	UPDATE profiles
	SET is_active = $1, updated_at = CURRENT_TIMESTAMP
	WHERE id = $2
	`

	_, err := ps.db.Exec(query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update profile status: %w", err)
	}

	log.Printf("✅ Profile %s status updated to %v", id, isActive)
	return nil
}

// DeleteProfile removes a profile from the database
func (ps *ProfileStorage) DeleteProfile(id string) error {
	query := `DELETE FROM profiles WHERE id = $1`

	_, err := ps.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	log.Printf("✅ Profile deleted: %s", id)
	return nil
}
