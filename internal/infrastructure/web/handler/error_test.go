package handler

import (
	"errors"
	"testing"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/pkg/apperror"
)

// TestDomainSentinelsPopulated guards against silently losing the init()
// registration that shares the sentinel list with pkg/telemetry.
func TestDomainSentinelsPopulated(t *testing.T) {
	if len(apperror.DomainSentinels) == 0 {
		t.Fatal("apperror.DomainSentinels should be populated after importing handler")
	}

	found := false
	for _, s := range apperror.DomainSentinels {
		if errors.Is(s, userdomain.ErrUserNotFound) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected userdomain.ErrUserNotFound to be registered in apperror.DomainSentinels")
	}
}
