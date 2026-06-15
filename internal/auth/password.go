// Package auth provides authentication and authorization services for the application.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
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

type parsedPasswordHash struct {
	params PasswordParams
	salt   []byte
	hash   []byte
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
		parsed.salt,
		parsed.params.Iterations,
		parsed.params.MemoryKiB,
		parsed.params.Parallelism,
		uint32(len(parsed.hash)),
	)

	if subtle.ConstantTimeCompare(parsed.hash, hash) != 1 {
		return domain.ErrUnauthenticated
	}

	return nil
}

func validatePasswordParams(params PasswordParams) error {
	if params.MemoryKiB < 19*1024 {
		return fmt.Errorf("argon2 memory must be at least 19456 KiB: %w", domain.ErrInvalidArgument)
	}

	if params.Iterations < 2 {
		return fmt.Errorf("argon2 iterations must be at least 2: %w", domain.ErrInvalidArgument)
	}

	if params.Parallelism < 1 {
		return fmt.Errorf("argon2 parallelism must be at least 1: %w", domain.ErrInvalidArgument)
	}

	if params.SaltBytes < minSaltBytes {
		return fmt.Errorf("argon2 salt must be at least %d bytes: %w", minSaltBytes, domain.ErrInvalidArgument)
	}

	if params.KeyBytes < minKeyBytes {
		return fmt.Errorf("argon2 key must be at least %d bytes: %w", minKeyBytes, domain.ErrInvalidArgument)
	}

	return nil
}

func randomBytes(size uint32) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}

	return buf, nil
}

func encodePasswordHash(params PasswordParams, salt, hash []byte) string {
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		encodedSalt,
		encodedHash,
	)
}

func parsePasswordHash(encodedHash string) (parsedPasswordHash, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" {
		return parsedPasswordHash{}, fmt.Errorf("password hash format is invalid: %w", domain.ErrInvalidArgument)
	}

	if parts[1] != "argon2id" {
		return parsedPasswordHash{}, fmt.Errorf("password hash algorithm is unsupported: %w", domain.ErrInvalidArgument)
	}

	version, err := parseVersion(parts[2])
	if err != nil {
		return parsedPasswordHash{}, err
	}

	if version != argon2.Version {
		return parsedPasswordHash{}, fmt.Errorf("password hash version is unsupported: %w", domain.ErrInvalidArgument)
	}

	params, err := parseParams(parts[3])
	if err != nil {
		return parsedPasswordHash{}, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return parsedPasswordHash{}, fmt.Errorf("password hash salt is invalid: %w", domain.ErrInvalidArgument)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return parsedPasswordHash{}, fmt.Errorf("password hash value is invalid: %w", domain.ErrInvalidArgument)
	}

	params.SaltBytes = uint32(len(salt))
	params.KeyBytes = uint32(len(hash))

	if err := validatePasswordParams(params); err != nil {
		return parsedPasswordHash{}, err
	}

	return parsedPasswordHash{
		params: params,
		salt:   salt,
		hash:   hash,
	}, nil
}

func parseVersion(raw string) (int, error) {
	value, ok := strings.CutPrefix(raw, "v=")
	if !ok {
		return 0, fmt.Errorf("password hash version is invalid: %w", domain.ErrInvalidArgument)
	}

	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("password hash version is invalid: %w", domain.ErrInvalidArgument)
	}

	return version, nil
}

func parseParams(raw string) (PasswordParams, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		return PasswordParams{}, fmt.Errorf("password hash params are invalid: %w", domain.ErrInvalidArgument)
	}

	values := make(map[string]string, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok || key == "" || value == "" {
			return PasswordParams{}, fmt.Errorf("password hash params are invalid: %w", domain.ErrInvalidArgument)
		}

		values[key] = value
	}

	memory, err := parseUint32Param(values, "m")
	if err != nil {
		return PasswordParams{}, err
	}

	iterations, err := parseUint32Param(values, "t")
	if err != nil {
		return PasswordParams{}, err
	}

	parallelism, err := parseUint8Param(values, "p")
	if err != nil {
		return PasswordParams{}, err
	}

	return PasswordParams{
		MemoryKiB:   memory,
		Iterations:  iterations,
		Parallelism: parallelism,
	}, nil
}

func parseUint32Param(values map[string]string, key string) (uint32, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("password hash param %q is missing: %w", key, domain.ErrInvalidArgument)
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("password hash param %q is invalid: %w", key, domain.ErrInvalidArgument)
	}

	return uint32(value), nil
}

func parseUint8Param(values map[string]string, key string) (uint8, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("password hash param %q is missing: %w", key, domain.ErrInvalidArgument)
	}

	value, err := strconv.ParseUint(raw, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("password hash param %q is invalid: %w", key, domain.ErrInvalidArgument)
	}

	return uint8(value), nil
}
