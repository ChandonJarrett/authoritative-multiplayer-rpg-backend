package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:  "valid",
			input: "correct horse battery staple",
		},
		{
			name:    "blank",
			input:   "   ",
			wantErr: domain.ErrInvalidArgument,
		},
		{
			name:    "too short",
			input:   "short",
			wantErr: domain.ErrInvalidArgument,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", MaxPasswordBytes+1),
			wantErr: domain.ErrInvalidArgument,
		},
		{
			name:  "exact min",
			input: strings.Repeat("a", MinPasswordBytes),
		},
		{
			name:  "exact max",
			input: strings.Repeat("a", MaxPasswordBytes),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	password := "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	if hash == password {
		t.Fatal("hash must not equal plaintext password")
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=1$") {
		t.Fatalf("unexpected hash format: %q", hash)
	}

	if err := VerifyPassword(hash, password); err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
}

func TestHashPasswordGeneratesDifferentHashes(t *testing.T) {
	password := "correct horse battery staple"

	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword first failed: %v", err)
	}

	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword second failed: %v", err)
	}

	if first == second {
		t.Fatal("expected different hashes because each hash must use a unique salt")
	}
}

func TestVerifyPasswordWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	err = VerifyPassword(hash, "wrong horse battery staple")
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestVerifyPasswordMissingHash(t *testing.T) {
	err := VerifyPassword("", "correct horse battery staple")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestVerifyPasswordMalformedHash(t *testing.T) {
	err := VerifyPassword("not-a-real-hash", "correct horse battery staple")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestHashPasswordWithParamsRejectsWeakParams(t *testing.T) {
	_, err := HashPasswordWithParams("correct horse battery staple", PasswordParams{
		MemoryKiB:   1024,
		Iterations:  1,
		Parallelism: 1,
		SaltBytes:   16,
		KeyBytes:    32,
	})

	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
