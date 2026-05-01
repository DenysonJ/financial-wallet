package apperror

import "sync"

// DomainSentinels is the canonical list of expected domain errors (4xx-like).
//
// It is populated at init() time by packages that own the domain-to-HTTP
// translation (today, internal/infrastructure/web/handler). The list is read
// by pkg/telemetry.IsExpected to decide whether a span should be marked as
// Error (unexpected failure, e.g. timeout) or kept Ok with a semantic
// attribute (expected business outcome, e.g. not-found).

var (
	sentinelsMu     sync.RWMutex
	DomainSentinels []error
)

// Register appends the given sentinel errors to DomainSentinels. It is
// intended to be called from init() blocks but is safe to call concurrently.
func Register(errs ...error) {
	sentinelsMu.Lock()
	defer sentinelsMu.Unlock()
	DomainSentinels = append(DomainSentinels, errs...)
}

// Sentinels returns a copy of the current sentinel list
func Sentinels() []error {
	sentinelsMu.RLock()
	defer sentinelsMu.RUnlock()
	out := make([]error, len(DomainSentinels))
	copy(out, DomainSentinels)
	return out
}
