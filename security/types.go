package security

import (
	"context"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/errs"
)

// Session interface defines multiple methods for session management.
type Session interface {
	// Set assigns a value to a session key. The context is typically used for request-scoped values.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('context.Context')
	// - key: the key under which the value is stored ('string')
	// - val: the value to store, which can be of any type ('any')
	// Returns:
	// - error: error, if any occurred while setting the value
	Set(ctx context.Context, key string, val any) error

	// Get retrieves the value associated with the key from the session.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('context.Context')
	// - key: the key for the value to be retrieved ('string')
	// Returns:
	// - mist.AnyValue: a wrapper containing the retrieved value or an error if the key wasn't found
	Get(ctx context.Context, key string) mist.AnyValue

	// Del deletes the key-value pair associated with the key from the session.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('context.Context')
	// - key: the key for the value to be deleted ('string')
	// Returns:
	// - error: error, if any occurred while deleting the value
	Del(ctx context.Context, key string) error

	// Destroy invalidates the session entirely, clearing all data within the session.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('context.Context')
	// Returns:
	// - error: error, if any occurred while destroying the session
	Destroy(ctx context.Context) error

	// Claims retrieves the claims associated with the session. Claims usually contain user-related data, often in a JWT context.
	// Returns:
	// - Claims: a set of claims related to the session
	Claims() Claims
}

// Provider interface defines methods for session lifecycle management and JWT claim updates.
type Provider interface {
	// InitSession initializes a new session with the specified user ID, JWT data, and session data.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('mist.Context')
	// - uid: user ID for which the session is being created ('int64')
	// - jwtData: JWT token data (usually claims) to store with the session ('map[string]any')
	// - sessData: additional session-specific data to associate with the session ('map[string]any')
	// Returns:
	// - Session: the initialized session
	// - error: error, if any occurred while initializing the session
	InitSession(ctx *mist.Context, uid int64, jwtData map[string]any, sessData map[string]any) (Session, error)

	// Get retrieves the current session associated with the context.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('mist.Context')
	// Returns:
	// - Session: the current session
	// - error: error, if any occurred while retrieving the session
	Get(ctx *mist.Context) (Session, error)

	// UpdateClaims updates the claims associated with the current session.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('mist.Context')
	// - claims: a new set of claims to associate with the session ('Claims')
	// Returns:
	// - error: error, if any occurred while updating the claims
	UpdateClaims(ctx *mist.Context, claims Claims) error

	// RenewAccessToken renews the access token associated with the session.
	// Parameters:
	// - ctx: context for managing deadlines, cancel operation signals, and other request-scoped values ('mist.Context')
	// Returns:
	// - error: error, if any occurred while renewing the access token
	RenewAccessToken(ctx *mist.Context) error

	// The ClearToken function is designed to remove or invalidate a security or session token associated with the given context.
	//
	// Parameters:
	// ctx: A pointer to a mist.Context object, which holds contextual information for the function to operate within.
	//      The mist.Context might include various details like user information, request scope, or environmental settings.
	//
	// Return:
	// error: This function returns an error type. If the token clearing process fails for any reason (e.g., token doesn't exist,
	//        network issues, permission issues), the function will return a non-nil error indicating what went wrong.
	//        If the token clearing process is successful, it returns nil.
	ClearToken(ctx *mist.Context) error
}

// Claims structure holds the data associated with the session's JWT claims.
type Claims struct {
	UserID    int64          // User ID
	SessionID string         // Session ID
	Data      map[string]any // Additional data related to the claims
}

// Get retrieves the value associated with the key from the claims.
func (c Claims) Get(key string) mist.AnyValue {
	val, ok := c.Data[key]
	if !ok {
		return mist.AnyValue{Err: errs.ErrKeyNotFound(key)} // Return an error if the key is not found
	}
	return mist.AnyValue{Val: val} // Return the value if the key is found
}
