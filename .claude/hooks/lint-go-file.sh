#!/bin/bash
# PostToolUse[Edit|Write] — Go diagnostics via gopls + goimports
# Uses gopls-lsp toolchain: goimports for formatting/imports, gopls check for diagnostics
set -uo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Only check Go files
[[ "$FILE_PATH" != *.go ]] && exit 0
[[ ! -f "$FILE_PATH" ]] && exit 0

ISSUES=""
# Newline real + printf %s na saida: o %b interpretaria as barras invertidas do
# conteudo (caminhos do Windows, escapes dentro de literais Go no diff), o que
# corrompe a mensagem ou aborta o printf ("missing unicode digit for \u").
NL=$'\n'

# 1. Formatting + imports (goimports subsumes gofmt + organizes imports)
if command -v goimports &>/dev/null; then
  DIFF=$(goimports -d "$FILE_PATH" 2>/dev/null)
  if [ -n "$DIFF" ]; then
    ISSUES="goimports: $FILE_PATH needs formatting/import fixes. Apply with: goimports -w \"$FILE_PATH\"${NL}${DIFF}${NL}"
  fi
else
  DIFF=$(gofmt -d "$FILE_PATH" 2>/dev/null)
  if [ -n "$DIFF" ]; then
    ISSUES="gofmt: $FILE_PATH is not formatted. Apply with: gofmt -w \"$FILE_PATH\"${NL}${DIFF}${NL}"
  fi
fi

# 2. gopls diagnostics (type errors, unused imports, missing deps — richer than go vet)
if command -v gopls &>/dev/null; then
  PKG_DIR=$(dirname "$FILE_PATH")
  DIAG=$(timeout 10 gopls check "./$PKG_DIR" 2>/dev/null || true)
  if [ -n "$DIAG" ]; then
    ISSUES="${ISSUES}${NL}gopls diagnostics:${NL}${DIAG}${NL}"
  fi
fi

if [ -n "$ISSUES" ]; then
  printf '%s' "$ISSUES" >&2
  exit 2
fi

exit 0
