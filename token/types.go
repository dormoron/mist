package token

import (
	"context"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/kit"
)

// Session represents the session management interface.
// It allows for getting, setting, and deleting session values, as well as destroying the session and retrieving session claims.
type Session interface {
	// Set stores a key-value pair in the session.
	// Parameters:
	// - ctx: Context for managing request-scoped values, deadlines, and cancellations.
	// - key: The key under which the value is stored.
	// - val: The value to store.
	// Returns:
	// - error: Potential error that occurred while setting the value.
	Set(ctx context.Context, key string, val any) error

	// Get retrieves a value based on the key from the session.
	// Parameters:
	// - ctx: Context for managing request-scoped values, deadlines, and cancellations.
	// - key: The key under which the value is stored.
	// Returns:
	// - kit.AnyValue: The value retrieved from the session, wrapped in an AnyValue type for flexibility.
	Get(ctx context.Context, key string) kit.AnyValue

	// Del deletes a key-value pair from the session based on the key.
	// Parameters:
	// - ctx: Context for managing request-scoped values, deadlines, and cancellations.
	// - key: The key of the value to be deleted.
	// Returns:
	// - error: Potential error that occurred while deleting the value.
	Del(ctx context.Context, key string) error

	// Destroy removes the session entirely.
	// Parameters:
	// - ctx: Context for managing request-scoped values, deadlines, and cancellations.
	// Returns:
	// - error: Potential error that occurred while destroying the session.
	Destroy(ctx context.Context) error

	// Claims returns the claims associated with the session.
	// Returns:
	// - Claims: The claims related to the session.
	Claims() Claims
}

// Provider represents the session provider interface.
// It supports session initialization, retrieval, updating claims, and renewing the access token.
type Provider interface {
	// InitSession initializes a session for the user with the provided data.
	// Parameters:
	// - ctx: The request context that includes HTTP-specific information.
	// - uid: The user ID associated with the session.
	// - jwtData: JWT-related data to be stored in the session.
	// - sessData: General session data to be stored.
	// Returns:
	// - Session: The initialized session object.
	// - error: Potential error that occurred during session initialization.
	InitSession(ctx *mist.Context, uid int64, jwtData map[string]string, sessData map[string]any) (Session, error)

	// Get retrieves the current session from the context.
	// Parameters:
	// - ctx: The request context that includes HTTP-specific information.
	// Returns:
	// - Session: The current session object.
	// - error: Potential error that occurred during session retrieval.
	Get(ctx *mist.Context) (Session, error)

	// UpdateClaims updates the claims associated with the current session.
	// Parameters:
	// - ctx: The request context that includes HTTP-specific information.
	// - claims: The new claims to be applied to the session.
	// Returns:
	// - error: Potential error that occurred during claims update.
	UpdateClaims(ctx *mist.Context, claims Claims) error

	// RenewAccessToken renews the access token for the current session.
	// Parameters:
	// - ctx: The request context that includes HTTP-specific information.
	// Returns:
	// - error: Potential error that occurred during access token renewal.
	RenewAccessToken(ctx *mist.Context) error
}

// Claims represent the data and identification associated with a session.
type Claims struct {
	Uid  int64             // User ID associated with the session.
	SSID string            // Session ID.
	Data map[string]string // Arbitrary data stored as part of the session claims.
}

// Get retrieves a specific claim based on the provided key.
// Parameters:
// - key: The key identifying the claim to be retrieved.
// Returns:
// - kit.AnyValue: A wrapper for the retrieved value, containing either the value or an error if not found.
func (c Claims) Get(key string) kit.AnyValue {
	val, ok := c.Data[key]
	if !ok {
		return kit.AnyValue{Err: errs.ErrKeyNotFound(key)}
	}
	return kit.AnyValue{Val: val}
}
