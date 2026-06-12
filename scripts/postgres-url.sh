#!/usr/bin/env bash
set -euo pipefail

required() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "ERROR: ${name} is required" >&2
    exit 1
  fi
}

required POSTGRES_HOST
required POSTGRES_PORT
required POSTGRES_USER
required POSTGRES_PASSWORD
required POSTGRES_DB
required POSTGRES_SSLMODE

tmp="$(mktemp /tmp/postgres-url-XXXX.go)"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'GO'
package main

import (
    "fmt"
    "net"
    "net/url"
    "os"
)

func main() {
    u := url.URL{
        Scheme: "postgres",
        User:   url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
        Host:   net.JoinHostPort(os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT")),
        Path:   "/" + os.Getenv("POSTGRES_DB"),
    }

    q := u.Query()
    q.Set("sslmode", os.Getenv("POSTGRES_SSLMODE"))
    u.RawQuery = q.Encode()

    fmt.Println(u.String())
}
GO

go run "$tmp"