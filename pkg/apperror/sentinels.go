package apperror

// DomainSentinels is the canonical list of expected domain errors (4xx-like).
//
// It is populated at init() time by packages that own the domain-to-HTTP
// translation (today, internal/infrastructure/web/handler). The list is read
// by pkg/telemetry.IsExpected to decide whether a span should be marked as
// Error (unexpected failure, e.g. timeout) or kept Ok with a semantic
// attribute (expected business outcome, e.g. not-found).
//
// The slice is only appended to during process startup via Register, so no
// synchronization is required at read time.
var DomainSentinels []error

// Register appends the given sentinel errors to DomainSentinels. It is
// intended to be called from init() blocks.
func Register(errs ...error) {
	DomainSentinels = append(DomainSentinels, errs...)
}
