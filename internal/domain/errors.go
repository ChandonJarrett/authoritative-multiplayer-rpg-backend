// Package domain contains shared domain-level concepts used across services.
package domain

import "errors"

var (
	// ErrInvalidArgument is returned when caller-provided input is malformed or unacceptable.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrUnauthenticated is returned when authentication is missing or invalid.
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrPermissionDenied is returned when the caller is authenticated but not allowed.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrAlreadyExists is returned when creating a resource that already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when a valid request conflicts with current state.
	ErrConflict = errors.New("conflict")

	// ErrUnavailable is returned when a required dependency or target service is unavailable.
	ErrUnavailable = errors.New("unavailable")

	// ErrInternal is returned for unexpected server-side failures.
	ErrInternal = errors.New("internal error")
)
