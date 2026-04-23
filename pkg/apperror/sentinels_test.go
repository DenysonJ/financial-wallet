package apperror_test

import (
	"errors"
	"testing"

	"github.com/DenysonJ/financial-wallet/pkg/apperror"
)

func TestRegister_AppendsToDomainSentinels(t *testing.T) {
	before := len(apperror.DomainSentinels)
	// Restore the registry so test-only sentinels don't leak into later tests
	// that assert the handler-populated domain set.
	t.Cleanup(func() { apperror.DomainSentinels = apperror.DomainSentinels[:before] })

	sentinelA := errors.New("test sentinel A")
	sentinelB := errors.New("test sentinel B")

	apperror.Register(sentinelA, sentinelB)

	if got := len(apperror.DomainSentinels); got != before+2 {
		t.Fatalf("expected %d sentinels after Register, got %d", before+2, got)
	}

	foundA, foundB := false, false
	for _, s := range apperror.DomainSentinels {
		if errors.Is(s, sentinelA) {
			foundA = true
		}
		if errors.Is(s, sentinelB) {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("registered sentinels missing: foundA=%v foundB=%v", foundA, foundB)
	}
}
