package token

import "github.com/dormoron/mist"

// CtxSessionKey is the key under which session information is stored in the context.
const CtxSessionKey = "_session"

// defaultProvider holds the default session provider that handles session operations.
var defaultProvider Provider

// InitSession initializes a session using the default session provider.
// Parameters:
// - ctx: The context associated with the current request.
// - uid: The user ID for which the session is being initialized.
// - jwtData: Map containing JWT-related data for the session.
// - sessData: Additional session data as a map.
// Returns:
// - Session: The newly initialized session object.
// - error: Potential error during session initialization.
func InitSession(ctx *mist.Context, uid int64,
	jwtData map[string]string,
	sessData map[string]any) (Session, error) {
	return defaultProvider.InitSession(
		ctx,
		uid,
		jwtData,
		sessData)
}

// Get retrieves the current session from the defaultProvider.
// Parameters:
// - ctx: The context associated with the current request.
// Returns:
// - Session: The session object retrieved.
// - error: Potential error during session retrieval.
func Get(ctx *mist.Context) (Session, error) {
	return defaultProvider.Get(ctx)
}

// SetDefaultProvider sets the provided session provider as the default provider.
// Parameters:
// - sp: The session provider to set as default.
func SetDefaultProvider(sp Provider) {
	defaultProvider = sp
}

// DefaultProvider returns the current default session provider.
// Returns:
// - Provider: The default session provider.
func DefaultProvider() Provider {
	return defaultProvider
}

// CheckLoginMiddleware creates and returns a middleware that uses the default session provider
// to check if the user is logged in for each request.
// Returns:
// - mist.Middleware: The middleware function for checking login status.
func CheckLoginMiddleware() mist.Middleware {
	return (&MiddlewareBuilder{sp: defaultProvider}).Build()
}

// RenewAccessToken renews the access token for the session associated with the given context.
// Parameters:
// - ctx: The context associated with the current request.
// Returns:
// - error: Potential error during access token renewal.
func RenewAccessToken(ctx *mist.Context) error {
	return defaultProvider.RenewAccessToken(ctx)
}

// UpdateClaims updates the session claims for the session associated with the given context.
// Parameters:
// - ctx: The context associated with the current request.
// - claims: The new claims to set for the session.
// Returns:
// - error: Potential error during claims update.
func UpdateClaims(ctx *mist.Context, claims Claims) error {
	return defaultProvider.UpdateClaims(ctx, claims)
}
