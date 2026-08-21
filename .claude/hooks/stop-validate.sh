#!/bin/bash
# Stop — Post-implementation validation gate
# Blocks Claude from finishing when Go code changes fail basic quality checks.
# Tiers:
#   1st attempt  → build + fmt + vet + unit tests
#   2nd attempt  → build + fmt + vet only (faster retry)
#   3rd+ attempt → pass (avoid infinite loop)
#
# Swagger and lint are NOT checked here — run manually or via /validate.
set -uo pipefail

INPUT=$(cat)
STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')

# ── Loop breaker ────────────────────────────────────────────────────
COUNTER_FILE="/tmp/claude-validate-${SESSION_ID}"
COUNT=$(cat "$COUNTER_FILE" 2>/dev/null || echo "0")
COUNT=$((COUNT + 1))
echo "$COUNT" > "$COUNTER_FILE"

if [ "$COUNT" -ge 3 ]; then
  rm -f "$COUNTER_FILE"
  exit 0
fi

# ── Skip during active Ralph Loop (intermediate iterations) ───────
SPECS_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)/.specs"
if find "$SPECS_DIR" -name "*.active.md" -type f 2>/dev/null | head -1 | grep -q .; then
  exit 0
fi

# ── Detect Go changes ──────────────────────────────────────────────
CHANGED_FILES=""
CHANGED_FILES+=$(git diff --name-only 2>/dev/null || true)
CHANGED_FILES+=$'\n'
CHANGED_FILES+=$(git diff --cached --name-only 2>/dev/null || true)
CHANGED_FILES+=$'\n'
CHANGED_FILES+=$(git ls-files --others --exclude-standard 2>/dev/null || true)

GO_CHANGES=$(echo "$CHANGED_FILES" | grep '\.go$' | sort -u || true)

# No Go changes → pass
if [ -z "$GO_CHANGES" ]; then
  rm -f "$COUNTER_FILE"
  exit 0
fi

ERRORS=""
# Newline real: as mensagens sao montadas com quebras de linha de verdade e
# impressas com %s. Usar escapes "\n" + printf %b quebra a saida, porque o %b
# tambem interpreta as barras invertidas do conteudo (ex.: caminhos do Windows
# como internal\usecases\... viram escapes invalidos e abortam o printf).
NL=$'\n'

# ── 1. Build ───────────────────────────────────────────────────────
BUILD_OUT=$(go build ./... 2>&1) || ERRORS="BUILD FAILED:${NL}${BUILD_OUT}${NL}${NL}"

# ── 2. Formatting (goimports > gofmt) ──────────────────────────────
if command -v goimports &>/dev/null; then
  FMT_FILES=$(goimports -l . 2>/dev/null | head -20)
  FMT_CMD="goimports -w ."
else
  FMT_FILES=$(gofmt -l . 2>/dev/null | head -20)
  FMT_CMD="gofmt -w ."
fi
if [ -n "$FMT_FILES" ]; then
  ERRORS="${ERRORS}FILES NOT FORMATTED (run ${FMT_CMD}):${NL}${FMT_FILES}${NL}${NL}"
fi

# ── 3. Go vet ──────────────────────────────────────────────────────
VET_OUT=$(go vet ./... 2>&1) || ERRORS="${ERRORS}GO VET ISSUES:${NL}${VET_OUT}${NL}${NL}"

# ── 4. Unit tests (first attempt only, skip on retry) ──────────────
if [ "$STOP_HOOK_ACTIVE" != "true" ] && [ -z "$ERRORS" ]; then
  TEST_OUT=$(go test ./internal/... -count=1 -short -timeout 60s 2>&1) || \
    ERRORS="${ERRORS}TEST FAILURES:${NL}${TEST_OUT}${NL}${NL}"
fi

# ── Result ──────────────────────────────────────────────────────────
if [ -n "$ERRORS" ]; then
  printf 'Post-implementation validation FAILED:\n\n%s' "$ERRORS" >&2
  exit 2
fi

# All passed
rm -f "$COUNTER_FILE"
exit 0
