// Package validate provides functions for validating and normalizing user input.
package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// Validation rules and patterns for user input.
const (
	MaxEmailBytes        = 320
	MinCharacterNameRune = 3
	MaxCharacterNameRune = 24
)

// Unicode-aware: letters + numbers + common separators
var characterNamePattern = regexp.MustCompile(`^[\p{L}\p{N} ._'-]+$`)

// normalize ensures consistent Unicode representation (NFC)
func normalize(s string) string {
	return norm.NFC.String(s)
}

// Email validates and normalizes an email address.
func Email(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", fmt.Errorf("email is required: %w", domain.ErrInvalidArgument)
	}

	// Normalize domain to ASCII (IDN support)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
	}

	domainASCII, err := idna.ToASCII(parts[1])
	if err != nil {
		return "", fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
	}

	email = parts[0] + "@" + domainASCII

	if len([]byte(email)) > MaxEmailBytes {
		return "", fmt.Errorf("email is too long: %w", domain.ErrInvalidArgument)
	}

	// Proper RFC-style validation
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
	}

	return email, nil
}

// CharacterName validates and normalizes a character name.
func CharacterName(name string) (string, error) {
	name = normalize(strings.TrimSpace(name))

	if name == "" {
		return "", fmt.Errorf("character name is required: %w", domain.ErrInvalidArgument)
	}

	runes := []rune(name)

	if len(runes) < MinCharacterNameRune {
		return "", fmt.Errorf("character name is too short: %w", domain.ErrInvalidArgument)
	}

	if len(runes) > MaxCharacterNameRune {
		return "", fmt.Errorf("character name is too long: %w", domain.ErrInvalidArgument)
	}

	if !characterNamePattern.MatchString(name) {
		return "", fmt.Errorf("character name contains unsupported characters: %w", domain.ErrInvalidArgument)
	}

	return name, nil
}

// RequiredID validates that a required ID field is not empty and trims whitespace.
func RequiredID(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required: %w", field, domain.ErrInvalidArgument)
	}
	return value, nil
}
