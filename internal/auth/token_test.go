package auth

import (
	"errors"
	"regexp"
	"testing"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestNewOpaqueToken(t *testing.T) {
	token, err := NewOpaqueToken(32)
	if err != nil {
		t.Fatalf("NewOpaqueToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected token")
	}

	if !tokenPattern.MatchString(token) {
		t.Fatalf("token is not URL-safe: %q", token)
	}
}

func TestNewOpaqueTokenDifferentEachCall(t *testing.T) {
	first, err := NewOpaqueToken(32)
	if err != nil {
		t.Fatalf("NewOpaqueToken first failed: %v", err)
	}

	second, err := NewOpaqueToken(32)
	if err != nil {
		t.Fatalf("NewOpaqueToken second failed: %v", err)
	}

	if first == second {
		t.Fatal("expected different tokens")
	}
}

func TestNewOpaqueTokenRejectsLowEntropy(t *testing.T) {
	_, err := NewOpaqueToken(8)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestNewSessionToken(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected token")
	}
}

func TestNewJoinToken(t *testing.T) {
	token, err := NewJoinToken()
	if err != nil {
		t.Fatalf("NewJoinToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected token")
	}
}
