package security

import (
	"github.com/dormoron/mist"
)

// CtxSessionKey is a constant string used as a key for storing session data in the context.
const CtxSessionKey = "_session"

// defaultProvider is a global variable that holds the default session provider.
var defaultProvider Provider

// InitSession initializes a new session using the default provider.
// Parameters:
// - ctx: The request context (*mist.Context).
// - uid: User ID for the session (int64).
// - jwtData: JWT-related data to be included in the session (map[string]any).
// - sessData: Additional session data (map[string]any).
// Returns:
// - Session: The newly created session.
// - error: An error if the session creation fails.
func InitSession(ctx *mist.Context, uid int64, jwtData map[string]any, sessData map[string]any) (Session, error) {
	// Use the default provider to initialize the session.
	return defaultProvider.InitSession(ctx, uid, jwtData, sessData)
}

// Get retrieves the session associated with the given context using the default provider.
// Parameters:
// - ctx: The request context (*mist.Context).
// Returns:
// - Session: The session associated with the context.
// - error: An error if the session retrieval fails.
func Get(ctx *mist.Context) (Session, error) {
	// Use the default provider to get the session from the context.
	return defaultProvider.Get(ctx)
}

// SetDefaultProvider sets the default session provider.
// Parameters:
// - sp: The session provider to be set as the default (Provider).
func SetDefaultProvider(sp Provider) {
	// Assign the provided session provider to the default provider variable.
	defaultProvider = sp
}

// DefaultProvider returns the current default session provider.
// Returns:
// - Provider: The current default session provider.
func DefaultProvider() Provider {
	// Return the default session provider.
	return defaultProvider
}

// CheckLoginMiddleware creates a middleware that checks if the user is logged in for specified paths.
// Parameters:
// - paths: A variadic list of URL paths to be checked (string).
// Returns:
// - mist.Middleware: The constructed middleware.
func CheckLoginMiddleware(paths ...string) mist.Middleware {
	// Initialize a MiddlewareBuilder with the default provider and specified paths,
	// and then build the middleware.
	return InitMiddlewareBuilder(defaultProvider, paths...).Build()
}

// RenewAccessToken renews the access token for the session associated with the given context.
// Parameters:
// - ctx: The request context (*mist.Context).
// Returns:
// - error: An error if the token renewal fails.
func RenewAccessToken(ctx *mist.Context) error {
	// Use the default provider to renew the access token.
	return defaultProvider.RenewAccessToken(ctx)
}

// ClearToken is a function that serves as a wrapper to invoke the ClearToken method
// of the defaultProvider. It clears the access and refresh tokens for a session by leveraging
// the default session provider.
//
// Parameters:
//   - ctx: The mist.Context object representing the current HTTP request and response.
//
// Returns:
//   - An error object if the underlying ClearToken method in defaultProvider fails, otherwise it returns nil.
func ClearToken(ctx *mist.Context) error {
	// Call the ClearToken method on the defaultProvider, passing in the context of the current HTTP request and response.
	// This delegates the task of clearing the session tokens to the defaultProvider.
	return defaultProvider.ClearToken(ctx)
}

// UpdateClaims updates the claims for the session associated with the given context.
// Parameters:
// - ctx: The request context (*mist.Context).
// - claims: The claims to be updated (Claims).
// Returns:
// - error: An error if the claims update fails.
func UpdateClaims(ctx *mist.Context, claims Claims) error {
	// Use the default provider to update the claims.
	return defaultProvider.UpdateClaims(ctx, claims)
}
