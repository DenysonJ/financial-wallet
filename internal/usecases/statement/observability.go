package statement

import (
	"context"

	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

const TracerKey = "usecase"

// Statement-specific action constants. They extend the generic CRUD set in
// pkg/logutil (ActionCreate, ActionList, ActionGet, ...) with operations that
// only apply to the statement domain. Kept local on purpose — ActionReverse
// and ActionImport are not meaningful outside financial statements
const (
	ActionReverse        = "reverse"
	ActionImport         = "import"
	ActionUpdateCategory = "update_category"
	ActionReplaceTags    = "replace_tags"
)

// injectLogContext enriches the context with structured logging fields for the use case layer.
func injectLogContext(ctx context.Context, action string) context.Context {
	return logutil.WithContext(ctx, logutil.StepUseCase, "statement", action)
}
