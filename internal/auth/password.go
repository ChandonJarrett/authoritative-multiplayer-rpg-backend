// Package auth provides authentication and authorization services for the application.
package auth

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"

	"golang.org/x/crypto/argon2"
)

const (
	// MinPasswordBytes is the minimum accepted plaintext password length.
	MinPasswordBytes = 12

	// MaxPasswordBytes caps password size to reduce accidental abuse and login DoS risk.
	MaxPasswordBytes = 1024

	minSaltBytes = 16
	minKeyBytes  = 32
)

// PasswordParams controls Argon2id password hashing cost.
type PasswordParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltBytes   uint32
	KeyBytes    uint32
}

// DefaultPasswordParams is the production default for new password hashes.
var DefaultPasswordParams = PasswordParams{
	MemoryKiB:   64 * 1024,
	Iterations:  3,
	Parallelism: 1,
	SaltBytes:   16,
	KeyBytes:    32,
}

// ParsedPasswordHash holds the components of a decoded Argon2id password hash.
type ParsedPasswordHash struct {
	Params PasswordParams
	Salt   []byte
	Hash   []byte
}

// ValidatePassword validates a plaintext password before hashing.
func ValidatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required: %w", domain.ErrInvalidArgument)
	}

	passwordBytes := len([]byte(password))
	if passwordBytes < MinPasswordBytes {
		return fmt.Errorf("password must be at least %d bytes: %w", MinPasswordBytes, domain.ErrInvalidArgument)
	}

	if passwordBytes > MaxPasswordBytes {
		return fmt.Errorf("password must be at most %d bytes: %w", MaxPasswordBytes, domain.ErrInvalidArgument)
	}

	return nil
}

// HashPassword validates and hashes a plaintext password with DefaultPasswordParams.
func HashPassword(password string) (string, error) {
	return HashPasswordWithParams(password, DefaultPasswordParams)
}

// HashPasswordWithParams validates and hashes a plaintext password with explicit Argon2id params.
func HashPasswordWithParams(password string, params PasswordParams) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	if err := validatePasswordParams(params); err != nil {
		return "", err
	}

	salt, err := randomBytes(params.SaltBytes)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.MemoryKiB,
		params.Parallelism,
		params.KeyBytes,
	)

	return encodePasswordHash(params, salt, hash), nil
}

// VerifyPassword verifies a plaintext password against a stored Argon2id hash.
func VerifyPassword(encodedHash, password string) error {
	if strings.TrimSpace(encodedHash) == "" {
		return fmt.Errorf("password hash is required: %w", domain.ErrInvalidArgument)
	}

	parsed, err := parsePasswordHash(encodedHash)
	if err != nil {
		return err
	}

	hash := argon2.IDKey(
		[]byte(password),
		parsed.Salt,
		parsed.Params.Iterations,
		parsed.Params.MemoryKiB,
		parsed.Params.Parallelism,
		uint32(len(parsed.Hash)), // #nosec G115 -- hash from base64 decode, always small
	)

	if subtle.ConstantTimeCompare(parsed.Hash, hash) != 1 {
		return domain.ErrUnauthenticated
	}

	return nil
}
