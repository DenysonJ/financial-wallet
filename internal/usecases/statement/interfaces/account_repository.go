package interfaces

import (
	"context"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// AccountRepository defines the contract for Account reads needed by statement use cases.
type AccountRepository interface {
	// FindByID returns an Account by its ID.
	// Returns account.ErrAccountNotFound if not found.
	FindByID(ctx context.Context, id vo.ID) (*accountdomain.Account, error)
}
