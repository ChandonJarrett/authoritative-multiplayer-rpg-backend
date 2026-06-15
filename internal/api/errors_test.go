package api

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

func TestToConnectErrorNil(t *testing.T) {
	if err := ToConnectError(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestToConnectErrorMappings(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code connect.Code
		msg  string
	}{
		{
			name: "invalid argument",
			err:  fmt.Errorf("email: %w", domain.ErrInvalidArgument),
			code: connect.CodeInvalidArgument,
			msg:  "invalid argument",
		},
		{
			name: "unauthenticated",
			err:  domain.ErrUnauthenticated,
			code: connect.CodeUnauthenticated,
			msg:  "unauthenticated",
		},
		{
			name: "permission denied",
			err:  domain.ErrPermissionDenied,
			code: connect.CodePermissionDenied,
			msg:  "permission denied",
		},
		{
			name: "already exists",
			err:  domain.ErrAlreadyExists,
			code: connect.CodeAlreadyExists,
			msg:  "already exists",
		},
		{
			name: "not found",
			err:  domain.ErrNotFound,
			code: connect.CodeNotFound,
			msg:  "not found",
		},
		{
			name: "conflict",
			err:  domain.ErrConflict,
			code: connect.CodeFailedPrecondition,
			msg:  "conflict",
		},
		{
			name: "unavailable",
			err:  domain.ErrUnavailable,
			code: connect.CodeUnavailable,
			msg:  "unavailable",
		},
		{
			name: "internal fallback",
			err:  errors.New("password leaked detail should not reach client"),
			code: connect.CodeInternal,
			msg:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ToConnectError(tt.err)
			if err == nil {
				t.Fatal("expected error")
			}

			if got := connect.CodeOf(err); got != tt.code {
				t.Fatalf("expected code %s, got %s", tt.code, got)
			}

			ce, ok := err.(*connect.Error)
			if !ok {
				t.Fatalf("expected *connect.Error, got %T", err)
			}

			if got := ce.Message(); got != tt.msg {
				t.Fatalf("expected message %q, got %q", tt.msg, got)
			}
		})
	}
}
