package statement

import (
	"context"

	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

const TracerKey = "usecase"
const ActionReverse = "reverse"
const ActionImport = "import"

// injectLogContext enriches the context with structured logging fields for the use case layer.
func injectLogContext(ctx context.Context, action string) context.Context {
	return logutil.WithContext(ctx, logutil.StepUseCase, "statement", action)
}
