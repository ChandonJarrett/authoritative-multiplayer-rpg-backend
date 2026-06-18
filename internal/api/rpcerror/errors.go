// Package rpcerror provides utilities for mapping internal domain errors to stable ConnectRPC errors.
package rpcerror

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// ToConnectError maps internal domain errors to stable ConnectRPC errors.
// Internal error details are hidden from clients. Log server-side before returning when needed.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}

	code, message := connectCodeAndMessage(err)

	return connect.NewError(code, errors.New(message))
}

func connectCodeAndMessage(err error) (connect.Code, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return connect.CodeInvalidArgument, "invalid argument"
	case errors.Is(err, domain.ErrUnauthenticated):
		return connect.CodeUnauthenticated, "unauthenticated"
	case errors.Is(err, domain.ErrPermissionDenied):
		return connect.CodePermissionDenied, "permission denied"
	case errors.Is(err, domain.ErrAlreadyExists):
		return connect.CodeAlreadyExists, "already exists"
	case errors.Is(err, domain.ErrNotFound):
		return connect.CodeNotFound, "not found"
	case errors.Is(err, domain.ErrConflict):
		return connect.CodeFailedPrecondition, "conflict"
	case errors.Is(err, domain.ErrUnavailable):
		return connect.CodeUnavailable, "unavailable"
	default:
		return connect.CodeInternal, "internal error"
	}
}
