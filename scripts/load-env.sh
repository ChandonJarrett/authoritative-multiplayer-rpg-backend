#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "ERROR: source this script instead of executing it:"
  echo "  source scripts/load-env.sh"
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "ERROR: ${ENV_FILE} not found"
  echo "Run: make env-init"
  return 1
fi

while IFS= read -r line || [[ -n "${line}" ]]; do
  line="${line#"${line%%[![:space:]]*}"}"
  line="${line%"${line##*[![:space:]]}"}"

  [[ -z "${line}" || "${line}" == \#* ]] && continue

  if [[ ! "${line}" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
    echo "ERROR: invalid env line in ${ENV_FILE}: ${line}"
    return 1
  fi

  key="${line%%=*}"
  value="${line#*=}"

  if [[ "${value}" =~ ^\".*\"$ || "${value}" =~ ^\'.*\'$ ]]; then
    value="${value:1:${#value}-2}"
  fi

  if [[ "${LOAD_ENV_OVERRIDE:-0}" == "1" || -z "${!key+x}" ]]; then
    export "${key}=${value}"
  fi
done < "${ENV_FILE}"

echo "Loaded ${ENV_FILE}"