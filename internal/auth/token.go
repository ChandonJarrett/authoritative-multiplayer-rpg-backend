package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

const (
	// SessionTokenBytes gives 256 bits of entropy.
	SessionTokenBytes = 32

	// JoinTokenBytes gives 256 bits of entropy.
	JoinTokenBytes = 32
)

// NewOpaqueToken creates a URL-safe opaque token with numBytes of randomness.
func NewOpaqueToken(numBytes int) (string, error) {
	if numBytes < 16 {
		return "", fmt.Errorf("token entropy must be at least 16 bytes: %w", domain.ErrInvalidArgument)
	}

	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NewSessionToken creates an opaque session token.
func NewSessionToken() (string, error) {
	return NewOpaqueToken(SessionTokenBytes)
}

// NewJoinToken creates an opaque join token.
func NewJoinToken() (string, error) {
	return NewOpaqueToken(JoinTokenBytes)
}
