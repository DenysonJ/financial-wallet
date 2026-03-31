#!/bin/bash
# PostToolUse[Edit|Write] — Validate Goose migration files have Up + Down sections
set -uo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Only check SQL files in the migration directory
[[ "$FILE_PATH" != */migration/*.sql ]] && exit 0
[[ ! -f "$FILE_PATH" ]] && exit 0

ERRORS=""

grep -q '^-- +goose Up' "$FILE_PATH" || ERRORS="Missing '-- +goose Up' section.\n"
grep -q '^-- +goose Down' "$FILE_PATH" || ERRORS="${ERRORS}Missing '-- +goose Down' section (migrations must be reversible).\n"

if [ -n "$ERRORS" ]; then
  printf "Migration validation failed — %s:\n%b" "$(basename "$FILE_PATH")" "$ERRORS" >&2
  exit 2
fi

exit 0
