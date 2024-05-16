package token

import "github.com/dormoron/mist"

// Builder serves as a constructor for creating new sessions with custom attributes.
type Builder struct {
	ctx      *mist.Context     // The context associated with the request.
	uid      int64             // The user ID for whom the session will be created.
	jwtData  map[string]string // JWT data that might be needed for the session.
	sessData map[string]any    // Additional data to store within the session.
	sp       Provider          // The session provider which will manage the session.
}

// InitSessionBuilder initializes a new Builder with the provided context and user ID.
// Parameters:
// - ctx: The context associated with the current request.
// - uid: The user ID for which the session is being built.
// Returns:
// - *Builder: A pointer to the newly initialized Builder instance.
func InitSessionBuilder(ctx *mist.Context, uid int64) *Builder {
	return &Builder{
		ctx: ctx,             // Set the request context.
		uid: uid,             // Set the user ID.
		sp:  defaultProvider, // Use the default session provider.
	}
}

// SetProvider sets the session provider for the Builder to the provided provider.
// Parameters:
// - p: The session provider to use for this session.
// Returns:
// - *Builder: The Builder instance to allow for method chaining.
func (b *Builder) SetProvider(p Provider) *Builder {
	b.sp = p // Set the custom session provider.
	return b // Return the Builder instance for chaining.
}

// SetJwtData assigns custom JWT data to the Builder.
// Parameters:
// - data: The JWT data map to store in the session.
// Returns:
// - *Builder: The Builder instance to allow for method chaining.
func (b *Builder) SetJwtData(data map[string]string) *Builder {
	b.jwtData = data // Set the JWT data map.
	return b         // Return the Builder instance for chaining.
}

// SetSessData assigns custom session data to the Builder.
// Parameters:
// - data: The session data map to store in the session.
// Returns:
// - *Builder: The Builder instance to allow for method chaining.
func (b *Builder) SetSessData(data map[string]any) *Builder {
	b.sessData = data // Set the session data map.
	return b          // Return the Builder instance for chaining.
}

// Build creates a new session using the data and provider set in the Builder.
// Returns:
// - Session: The newly created session object.
// - error: Potential error encountered during session initialization.
func (b *Builder) Build() (Session, error) {
	// Use the session provider's InitSession method to create the session with all provided attributes.
	return b.sp.InitSession(b.ctx, b.uid, b.jwtData, b.sessData)
}
