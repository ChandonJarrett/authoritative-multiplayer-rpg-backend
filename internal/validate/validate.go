// Package validate provides input validation and normalization helpers.
package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

const (
	// MaxEmailBytes is the practical upper bound for an email address.
	MaxEmailBytes = 320

	// MinCharacterNameRunes is the minimum visible length for a character name.
	MinCharacterNameRunes = 3

	// MaxCharacterNameRunes is the maximum visible length for a character name.
	MaxCharacterNameRunes = 24

	maxIDBytes      = 64
	maxTokenBytes   = 128
	maxAddressBytes = 255
)

var characterNamePattern = regexp.MustCompile(`^[\p{L}\p{N} ._'\-]+$`)

// Email normalizes and validates an email address.
func Email(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", fmt.Errorf("email is required: %w", domain.ErrInvalidArgument)
	}

	local, rawDomain, ok := strings.Cut(email, "@")
	if !ok || local == "" || rawDomain == "" || strings.Contains(rawDomain, "@") {
		return "", fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
	}

	asciiDomain, err := idna.Lookup.ToASCII(rawDomain)
	if err != nil {
		return "", fmt.Errorf("email domain is invalid: %w", domain.ErrInvalidArgument)
	}

	email = local + "@" + asciiDomain
	if len([]byte(email)) > MaxEmailBytes {
		return "", fmt.Errorf("email must be at most %d bytes: %w", MaxEmailBytes, domain.ErrInvalidArgument)
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
	}

	return email, nil
}

// CharacterName normalizes and validates a character name.
func CharacterName(name string) (string, error) {
	name = normalizeText(strings.TrimSpace(name))
	if name == "" {
		return "", fmt.Errorf("character name is required: %w", domain.ErrInvalidArgument)
	}

	runeCount := utf8.RuneCountInString(name)
	if runeCount < MinCharacterNameRunes {
		return "", fmt.Errorf("character name must be at least %d characters: %w", MinCharacterNameRunes, domain.ErrInvalidArgument)
	}
	if runeCount > MaxCharacterNameRunes {
		return "", fmt.Errorf("character name must be at most %d characters: %w", MaxCharacterNameRunes, domain.ErrInvalidArgument)
	}
	if strings.ContainsAny(name, "\t\r\n") {
		return "", fmt.Errorf("character name must not contain control whitespace: %w", domain.ErrInvalidArgument)
	}
	if !characterNamePattern.MatchString(name) {
		return "", fmt.Errorf("character name contains unsupported characters: %w", domain.ErrInvalidArgument)
	}

	return name, nil
}

// UserID validates a required user identifier.
func UserID(value string) (string, error) {
	return requiredKeySegment("user ID", value, maxIDBytes)
}

// CharacterID validates a required character identifier.
func CharacterID(value string) (string, error) {
	return requiredKeySegment("character ID", value, maxIDBytes)
}

// GameServerID validates a required game server identifier.
func GameServerID(value string) (string, error) {
	return requiredKeySegment("game server ID", value, maxIDBytes)
}

// OptionalGameServerID validates an optional game server identifier.
// Empty input means the caller wants automatic server selection.
func OptionalGameServerID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	return GameServerID(value)
}

// SessionToken validates a required session token.
func SessionToken(value string) (string, error) {
	return requiredKeySegment("session token", value, maxTokenBytes)
}

// JoinToken validates a required game join token.
func JoinToken(value string) (string, error) {
	return requiredKeySegment("join token", value, maxTokenBytes)
}

// ServerAddress validates a required network address or advertised endpoint.
func ServerAddress(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("server address is required: %w", domain.ErrInvalidArgument)
	}
	if len([]byte(value)) > maxAddressBytes {
		return "", fmt.Errorf("server address must be at most %d bytes: %w", maxAddressBytes, domain.ErrInvalidArgument)
	}
	if strings.ContainsAny(value, "\t\r\n") {
		return "", fmt.Errorf("server address must not contain control whitespace: %w", domain.ErrInvalidArgument)
	}
	return value, nil
}

// RequiredID trims and validates a required identifier.
//
// Prefer domain-specific helpers above.
func RequiredID(field, value string) (string, error) {
	return requiredKeySegment(field, value, maxIDBytes)
}

func requiredKeySegment(field, value string, maxBytes int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required: %w", field, domain.ErrInvalidArgument)
	}
	if len([]byte(value)) > maxBytes {
		return "", fmt.Errorf("%s must be at most %d bytes: %w", field, maxBytes, domain.ErrInvalidArgument)
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return "", fmt.Errorf("%s must not contain whitespace: %w", field, domain.ErrInvalidArgument)
	}
	if strings.Contains(value, ":") {
		return "", fmt.Errorf("%s must not contain colon: %w", field, domain.ErrInvalidArgument)
	}
	return value, nil
}

func normalizeText(value string) string {
	return norm.NFC.String(value)
}
