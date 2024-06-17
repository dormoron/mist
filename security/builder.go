package security

import "github.com/dormoron/mist"

// Builder is a structure that helps in building a session configuration step by step.
// It contains the context, user ID, JWT data, session data, and a session provider.
type Builder struct {
	ctx      *mist.Context  // The request context.
	uid      int64          // The user ID for the session.
	jwtData  map[string]any // JWT-related data to be included in the session.
	sessData map[string]any // Additional session data.
	sp       Provider       // The session provider used to initialize the session.
}

// InitSessionBuilder initializes and returns a new instance of Builder with the given context and user ID.
// The default session provider is set during initialization.
// Parameters:
// - ctx: The request context (*mist.Context).
// - uid: The user ID for the session (int64).
// Returns:
// - *Builder: A pointer to a newly created Builder instance.
func InitSessionBuilder(ctx *mist.Context, uid int64) *Builder {
	return &Builder{
		ctx: ctx,
		uid: uid,
		sp:  defaultProvider, // Set the default session provider.
	}
}

// SetProvider sets a custom session provider for the Builder.
// Parameters:
// - p: The custom session provider (Provider).
// Returns:
// - *Builder: The Builder instance with the updated provider.
func (b *Builder) SetProvider(p Provider) *Builder {
	b.sp = p // Update the session provider.
	return b
}

// SetJwtData sets the JWT data for the Builder.
// Parameters:
// - data: The JWT-related data (map[string]any).
// Returns:
// - *Builder: The Builder instance with the updated JWT data.
func (b *Builder) SetJwtData(data map[string]any) *Builder {
	b.jwtData = data // Update the JWT data.
	return b
}

// SetSessData sets the session data for the Builder.
// Parameters:
// - data: The additional session data (map[string]any).
// Returns:
// - *Builder: The Builder instance with the updated session data.
func (b *Builder) SetSessData(data map[string]any) *Builder {
	b.sessData = data // Update the session data.
	return b
}

// Build constructs the session using the provided or default session provider, context, user ID, JWT data, and session data.
// Returns:
// - Session: The newly created session.
// - error: An error if the session creation fails.
func (b *Builder) Build() (Session, error) {
	return b.sp.InitSession(b.ctx, b.uid, b.jwtData, b.sessData)
	// Use the session provider to initialize the session with the given parameters.
}
