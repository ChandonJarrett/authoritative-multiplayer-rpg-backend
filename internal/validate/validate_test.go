package validate

import (
	"errors"
	"testing"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid normalized", input: " USER@Example.COM ", want: "user@example.com"},
		{name: "missing", input: " ", wantErr: true},
		{name: "invalid", input: "not-email", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Email(tt.input)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidArgument) {
					t.Fatalf("expected invalid argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil err, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCharacterName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: " Hero_01 ", want: "Hero_01"},
		{name: "too short", input: "ab", wantErr: true},
		{name: "unsupported", input: "hero!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CharacterName(tt.input)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidArgument) {
					t.Fatalf("expected invalid argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil err, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
