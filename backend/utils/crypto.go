package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// JWT secret key - in production, this should be stored in environment variables
	JWTSecret = "your-super-secret-jwt-key-change-in-production"
	// JWT expiration time
	JWTExpiration = 24 * time.Hour
	// Salt length for password hashing
	SaltLength = 32
)

// GenerateSalt generates a random salt for password hashing
func GenerateSalt() (string, error) {
	salt := make([]byte, SaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return hex.EncodeToString(salt), nil
}

// HashPassword creates a secure hash of the password using SHA256 and salt
func HashPassword(password, salt string) string {
	// Combine password and salt
	combined := password + salt
	// Create SHA256 hash
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword verifies a password against its hash and salt
func VerifyPassword(password, hash, salt string) bool {
	expectedHash := HashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(hash), []byte(expectedHash)) == 1
}

// GenerateJWT creates a JWT token for the user
func GenerateJWT(userID int, username, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(JWTExpiration).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      "mulic2",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ExtractUserFromJWT extracts user information from JWT claims
func ExtractUserFromJWT(claims jwt.MapClaims) (int, string, string, error) {
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", "", fmt.Errorf("invalid user_id in token")
	}

	username, ok := claims["username"].(string)
	if !ok {
		return 0, "", "", fmt.Errorf("invalid username in token")
	}

	role, ok := claims["role"].(string)
	if !ok {
		return 0, "", "", fmt.Errorf("invalid role in token")
	}

	return int(userID), username, role, nil
}

// GenerateSecureToken generates a secure random token for session management
func GenerateSecureToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(token), nil
}
