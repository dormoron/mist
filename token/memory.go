package token

import (
	"context"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/utils"
)

// Ensure MemorySession implements the Session interface.
var _ Session = &MemorySession{}

// MemorySession is an in-memory implementation of the Session interface.
// It stores session data and associated claims.
type MemorySession struct {
	data   map[string]any // Map to hold session key-value pairs.
	claims Claims         // Claims associated with the session.
}

// Destroy does nothing in MemorySession because there is no persistent store to clean up.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// Returns:
// - error: Always returns nil for MemorySession.
func (m *MemorySession) Destroy(ctx context.Context) error {
	return nil
}

// UpdateClaims is a stub method for MemorySession. It doesn't perform any operations as claims are not updated dynamically in this implementation.
// Parameters:
// - ctx: The request context with HTTP-specific information.
// - claims: The new claims data to store in the session.
// Returns:
// - error: Always returns nil for MemorySession.
func (m *MemorySession) UpdateClaims(ctx *mist.Context, claims Claims) error {
	return nil
}

// Del removes the value associated with the given key from the session.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key of the session data to delete.
// Returns:
// - error: Returns nil after successfully deleting the key-value pair, if any.
func (m *MemorySession) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// InitMemorySession initializes a new MemorySession with the given Claims.
// Parameters:
// - cl: Claims to associate with the new session.
// Returns:
// - *MemorySession: A pointer to the newly created MemorySession instance.
func InitMemorySession(cl Claims) *MemorySession {
	return &MemorySession{
		data:   map[string]any{}, // Initialize the data map.
		claims: cl,               // Set the provided Claims.
	}
}

// Set stores the provided key-value pair into the session's data map.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key for the session data to store.
// - val: The value to associate with the key.
// Returns:
// - error: Returns nil after adding or updating the key-value pair in the session's data map.
func (m *MemorySession) Set(ctx context.Context, key string, val any) error {
	m.data[key] = val
	return nil
}

// Get retrieves the value associated with the given key in the session's data map.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key of the session data to retrieve.
// Returns:
// - utils.AnyValue: A struct wrapping the retrieved value or an error if the key does not exist.
func (m *MemorySession) Get(ctx context.Context, key string) utils.AnyValue {
	val, ok := m.data[key]
	if !ok {
		return utils.AnyValue{Err: errs.ErrKeyNotFound(key)}
	}
	return utils.AnyValue{Val: val}
}

// Claims returns the Claims associated with the session.
// Returns:
// - Claims: The current set of claims stored with the session.
func (m *MemorySession) Claims() Claims {
	return m.claims
}
