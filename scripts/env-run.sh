#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${ENV_FILE:-.env}"

if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source scripts/load-env.sh "${ENV_FILE}" >/dev/null
fi

exec "$@"