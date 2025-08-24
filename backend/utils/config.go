package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() map[string]string {
	config := make(map[string]string)

	// Load from environment variables first
	envVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"PORT", "JWT_SECRET", "JWT_EXPIRATION", "CORS_ORIGIN",
		"MAX_LOGIN_ATTEMPTS", "ACCOUNT_LOCKOUT_DURATION",
	}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			config[envVar] = value
		}
	}

	// Load from config file if it exists
	configPath := getConfigPath()
	if configFromFile := loadConfigFile(configPath); len(configFromFile) > 0 {
		for key, value := range configFromFile {
			// Only set if not already set by environment variable
			if _, exists := config[key]; !exists {
				config[key] = value
			}
		}
	}

	// Set defaults for missing values
	setDefaults(config)

	return config
}

// getConfigPath returns the appropriate config file path for the current platform
func getConfigPath() string {
	// Look for config files in order of preference
	configFiles := []string{
		"config.env",
		".env",
		"config.txt",
	}

	// Check current directory first
	for _, filename := range configFiles {
		if _, err := os.Stat(filename); err == nil {
			return filename
		}
	}

	// Check backend directory
	backendDir := "backend"
	if _, err := os.Stat(backendDir); err == nil {
		for _, filename := range configFiles {
			path := filepath.Join(backendDir, filename)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Check parent directory
	parentDir := ".."
	if _, err := os.Stat(parentDir); err == nil {
		for _, filename := range configFiles {
			path := filepath.Join(parentDir, filename)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// loadConfigFile reads configuration from a file
func loadConfigFile(filepath string) map[string]string {
	config := make(map[string]string)

	file, err := os.Open(filepath)
	if err != nil {
		return config
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Remove quotes if present
				value = strings.Trim(value, `"'`)

				config[key] = value
			}
		}
	}

	return config
}

// setDefaults sets default values for missing configuration
func setDefaults(config map[string]string) {
	defaults := map[string]string{
		"DB_HOST":                  "localhost",
		"DB_PORT":                  "5432",
		"DB_USER":                  "postgres",
		"DB_PASSWORD":              "postgres",
		"DB_NAME":                  "mulic2_db",
		"PORT":                     "8080",
		"JWT_SECRET":               "your-super-secret-jwt-key-change-in-production",
		"JWT_EXPIRATION":           "24h",
		"CORS_ORIGIN":              "*",
		"MAX_LOGIN_ATTEMPTS":       "5",
		"ACCOUNT_LOCKOUT_DURATION": "15m",
	}

	for key, defaultValue := range defaults {
		if _, exists := config[key]; !exists {
			config[key] = defaultValue
		}
	}
}

// GetConfigValue retrieves a configuration value
func GetConfigValue(key string) string {
	config := LoadConfig()
	return config[key]
}

// GetConfigValueWithDefault retrieves a configuration value with a fallback default
func GetConfigValueWithDefault(key, defaultValue string) string {
	if value := GetConfigValue(key); value != "" {
		return value
	}
	return defaultValue
}

// PrintConfig prints the current configuration (without sensitive data)
func PrintConfig() {
	config := LoadConfig()

	fmt.Println("=== MuliC2 Configuration ===")
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Database Host: %s\n", config["DB_HOST"])
	fmt.Printf("Database Port: %s\n", config["DB_PORT"])
	fmt.Printf("Database Name: %s\n", config["DB_NAME"])
	fmt.Printf("Database User: %s\n", config["DB_USER"])
	fmt.Printf("Server Port: %s\n", config["PORT"])
	fmt.Printf("CORS Origin: %s\n", config["CORS_ORIGIN"])
	fmt.Printf("Max Login Attempts: %s\n", config["MAX_LOGIN_ATTEMPTS"])
	fmt.Printf("Account Lockout Duration: %s\n", config["ACCOUNT_LOCKOUT_DURATION"])
	fmt.Println("=============================")
}
