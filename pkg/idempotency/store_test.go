package idempotency

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoryStore is a minimal in-memory Store implementation used to
// verify the interface contract without any external dependency.
type memoryStore struct {
	mu      sync.Mutex
	entries map[string]*Entry
}

func newMemoryStore() *memoryStore {
	return &memoryStore{entries: make(map[string]*Entry)}
}

func (m *memoryStore) Lock(_ context.Context, key, fingerprint string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[key]; exists {
		return false, nil
	}

	m.entries[key] = &Entry{
		Status:      StatusProcessing,
		Fingerprint: fingerprint,
	}
	return true, nil
}

func (m *memoryStore) Get(_ context.Context, key string) (*Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil, nil
	}

	// Return a copy to avoid mutation of the internal state.
	cp := *entry
	return &cp, nil
}

func (m *memoryStore) Complete(_ context.Context, key string, entry *Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry.Status = StatusCompleted
	cp := *entry
	m.entries[key] = &cp
	return nil
}

func (m *memoryStore) Unlock(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
	return nil
}

// ---------- Interface contract tests ----------

func TestStore_LockAcquiresOnFirstCall(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	acquired, lockErr := store.Lock(ctx, "key-1", "fp-abc")

	require.NoError(t, lockErr)
	assert.True(t, acquired)
}

func TestStore_LockReturnsFalseOnSecondCall(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	acquired, lockErr := store.Lock(ctx, "key-1", "fp-abc")

	require.NoError(t, lockErr)
	assert.False(t, acquired, "second Lock for the same key must return false")
}

func TestStore_GetReturnsNilForNonExistentKey(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	entry, getErr := store.Get(ctx, "non-existent")

	require.NoError(t, getErr)
	assert.Nil(t, entry)
}

func TestStore_GetReturnsEntryAfterLock(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	entry, getErr := store.Get(ctx, "key-1")

	require.NoError(t, getErr)
	require.NotNil(t, entry)
	assert.Equal(t, StatusProcessing, entry.Status)
	assert.Equal(t, "fp-abc", entry.Fingerprint)
}

func TestStore_CompleteChangesStatusToCompleted(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	completeEntry := &Entry{
		StatusCode:  201,
		Body:        []byte(`{"id":"123"}`),
		Fingerprint: "fp-abc",
	}
	completeErr := store.Complete(ctx, "key-1", completeEntry)

	require.NoError(t, completeErr)
	assert.Equal(t, StatusCompleted, completeEntry.Status, "Complete must set Status on the entry")
}

func TestStore_GetReturnsCompletedEntryAfterComplete(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	completeEntry := &Entry{
		StatusCode:  201,
		Body:        []byte(`{"id":"123"}`),
		Fingerprint: "fp-abc",
	}
	_ = store.Complete(ctx, "key-1", completeEntry)

	entry, getErr := store.Get(ctx, "key-1")

	require.NoError(t, getErr)
	require.NotNil(t, entry)
	assert.Equal(t, StatusCompleted, entry.Status)
	assert.Equal(t, 201, entry.StatusCode)
	assert.Equal(t, []byte(`{"id":"123"}`), entry.Body)
	assert.Equal(t, "fp-abc", entry.Fingerprint)
}

func TestStore_UnlockRemovesEntry(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	unlockErr := store.Unlock(ctx, "key-1")
	require.NoError(t, unlockErr)

	entry, getErr := store.Get(ctx, "key-1")
	require.NoError(t, getErr)
	assert.Nil(t, entry, "entry must be nil after Unlock")
}

func TestStore_GetReturnsNilAfterUnlock(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	_, _ = store.Lock(ctx, "key-1", "fp-abc")

	_ = store.Complete(ctx, "key-1", &Entry{
		StatusCode:  200,
		Body:        []byte(`{}`),
		Fingerprint: "fp-abc",
	})

	_ = store.Unlock(ctx, "key-1")

	entry, getErr := store.Get(ctx, "key-1")

	require.NoError(t, getErr)
	assert.Nil(t, entry)
}

func TestStore_LockSucceedsAfterUnlock(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	// First lock
	acquired, _ := store.Lock(ctx, "key-1", "fp-abc")
	require.True(t, acquired)

	// Unlock (simulating a 5xx retry scenario)
	_ = store.Unlock(ctx, "key-1")

	// Re-lock must succeed
	acquired, lockErr := store.Lock(ctx, "key-1", "fp-def")

	require.NoError(t, lockErr)
	assert.True(t, acquired, "Lock must succeed after Unlock (retry pattern)")
}

func TestStore_IndependentKeys(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	acquired1, _ := store.Lock(ctx, "key-1", "fp-1")
	acquired2, _ := store.Lock(ctx, "key-2", "fp-2")

	assert.True(t, acquired1)
	assert.True(t, acquired2)

	// Unlocking key-1 must not affect key-2
	_ = store.Unlock(ctx, "key-1")

	entry, getErr := store.Get(ctx, "key-2")
	require.NoError(t, getErr)
	require.NotNil(t, entry, "key-2 must still exist after key-1 is unlocked")
	assert.Equal(t, StatusProcessing, entry.Status)
}

// Verify the memoryStore satisfies the Store interface at compile time.
var _ Store = (*memoryStore)(nil)
