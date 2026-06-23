// Package auth provides authentication and authorization services for the application.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"

	"golang.org/x/crypto/argon2"
)

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

func parsePasswordHash(encodedHash string) (ParsedPasswordHash, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" {
		return ParsedPasswordHash{}, fmt.Errorf("password hash format is invalid: %w", domain.ErrInvalidArgument)
	}

	if parts[1] != "argon2id" {
		return ParsedPasswordHash{}, fmt.Errorf("password hash algorithm is unsupported: %w", domain.ErrInvalidArgument)
	}

	version, err := parseVersion(parts[2])
	if err != nil {
		return ParsedPasswordHash{}, err
	}

	if version != argon2.Version {
		return ParsedPasswordHash{}, fmt.Errorf("password hash version is unsupported: %w", domain.ErrInvalidArgument)
	}

	params, err := parseParams(parts[3])
	if err != nil {
		return ParsedPasswordHash{}, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ParsedPasswordHash{}, fmt.Errorf("password hash salt is invalid: %w", domain.ErrInvalidArgument)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ParsedPasswordHash{}, fmt.Errorf("password hash value is invalid: %w", domain.ErrInvalidArgument)
	}

	params.SaltBytes = uint32(len(salt)) // #nosec G115 -- salt from base64 decode, always small
	params.KeyBytes = uint32(len(hash))  // #nosec G115 -- hash from base64 decode, always small

	if err := validatePasswordParams(params); err != nil {
		return ParsedPasswordHash{}, err
	}

	return ParsedPasswordHash{
		Params: params,
		Salt:   salt,
		Hash:   hash,
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
