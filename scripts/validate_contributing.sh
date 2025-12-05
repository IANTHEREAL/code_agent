#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FILE="$ROOT_DIR/CONTRIBUTING.md"

require_headings=(
  "Welcome & Introduction"
  "Prerequisites"
  "Development Setup"
  "Project Structure Overview"
  "Development Workflow"
  "Code Style & Best Practices"
  "Testing Guidelines"
  "Submitting Changes"
  "Communication & Questions"
)

if [[ ! -f "$FILE" ]]; then
  echo "validation failed: $FILE does not exist" >&2
  exit 1
fi

search_cmd=""
if command -v rg >/dev/null 2>&1; then
  search_cmd="rg -q"
else
  search_cmd="grep -q"
fi

missing=()
for heading in "${require_headings[@]}"; do
  pattern="^## ${heading}$"
  if ! eval "$search_cmd \"\$pattern\" \"$FILE\""; then
    missing+=("$heading")
  fi
done

if (( ${#missing[@]} > 0 )); then
  printf "validation failed: missing headings -> %s\n" "${missing[*]}" >&2
  exit 1
fi

echo "CONTRIBUTING.md validation passed."
