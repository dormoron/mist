package security

import (
	"net/http"

	"github.com/dormoron/mist"
)

// MiddlewareBuilder is a structure that helps build middleware for handling HTTP requests.
// It contains a provider for session management and a list of paths to be handled.
type MiddlewareBuilder struct {
	sp    Provider // The session provider, used to get session information.
	paths []string // A list of URL paths that the middleware will handle.
}

// InitMiddlewareBuilder initializes and returns a new instance of MiddlewareBuilder.
// Parameters:
// - sp: The session provider for session management (Provider).
// - paths: A variadic list of URL paths to be handled by the middleware (string).
// Returns:
// - *MiddlewareBuilder: A pointer to a newly created MiddlewareBuilder instance.
func InitMiddlewareBuilder(sp Provider, paths ...string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		sp:    sp,    // Set the session provider.
		paths: paths, // Set the list of paths.
	}
}

// Build constructs the middleware function.
// Returns:
// - mist.Middleware: A middleware function that processes the HTTP requests.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	// Return a middleware function that processes the HTTP request and context.
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// Check if the request URL path matches any of the specified paths.
			for _, path := range b.paths {
				if ctx.Request.URL.Path == path {
					next(ctx) // If a match is found, call the next handler and return.
					return
				}
			}

			// If no matching path is found, retrieve the session using the session provider.
			sess, err := b.sp.Get(ctx)
			if err != nil {
				// If there is an error getting the session, abort with an unauthorized status.
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			// Set the session information in the context.
			ctx.Set(CtxSessionKey, sess)
			next(ctx) // Call the next handler.
		}
	}
}
