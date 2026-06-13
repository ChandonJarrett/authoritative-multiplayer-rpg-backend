// Package testutil provides shared helpers for integration and unit tests.
package testutil

import (
	"fmt"
	"os"
	"testing"
)

// SkipOnServiceError skips the current test when a service dependency is
// unavailable. In CI (CI=true) it calls t.Fatal instead, because services
// are always expected to be reachable there.
func SkipOnServiceError(t *testing.T, err error, message string) {
	t.Helper()
	if inCI() {
		t.Fatalf("%s: %v", message, err)
	}
	t.Skipf("%s: %v (set CI=true to promote this to a failure)", message, err)
}

// SkipOnServiceErrorf is like SkipOnServiceError but accepts a fmt-style format
// string and arguments for the message.
func SkipOnServiceErrorf(t *testing.T, err error, format string, args ...any) {
	t.Helper()
	SkipOnServiceError(t, err, fmt.Sprintf(format, args...))
}

// RequireCI skips the test unless CI=true. Use this to mark tests that depend
// on infrastructure that is only guaranteed to be present in CI.
func RequireCI(t *testing.T) {
	t.Helper()
	if !inCI() {
		t.Skip("skipping: test requires CI environment (CI=true)")
	}
}

// inCI reports whether the process is running inside a CI environment.
func inCI() bool {
	return os.Getenv("CI") == "true"
}
