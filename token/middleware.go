package token

import (
	"github.com/dormoron/mist"
	"log"
	"log/slog"
	"net/http"
	"regexp"
)

// MiddlewareBuilder is a type that encapsulates session management within middleware.
//
// Fields:
// - sp: An instance of a Provider that manages session-related operations.
type MiddlewareBuilder struct {
	sp    Provider
	paths []*regexp.Regexp // A slice of pointer to regular expressions. Each pattern in this slice will be used
}

// Build constructs a middleware function that integrates session management.
// This middleware retrieves the session for each request and stores it in the context if successful.
// If the session cannot be obtained, the middleware logs an error and aborts the request with an HTTP 401 Unauthorized status.
//
// Returns:
// - mist.Middleware: A middleware function that handles session retrieval and management.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// Retrieve the path from the request URL.
			requestPath := ctx.Request.URL.Path

			// If the request's path matches any of the path patterns,
			// call the next mist.HandleFunc in the chain and return from the current one.
			for _, pattern := range b.paths {
				if pattern.MatchString(requestPath) {
					next(ctx)
					return
				}
			}
			// Attempt to retrieve the current session from the session provider.
			sess, err := b.sp.Get(ctx)
			if err != nil {
				// If there is an error retrieving the session, log the unauthorized access attempt and abort the request.
				slog.Debug("unauthorized", slog.Any("err", err))
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			// Successfully retrieved session, so set it in the request's context under a pre-defined session context key.
			ctx.Set(CtxSessionKey, sess)
			// Call the next middleware or final handler in the chain.
			next(ctx)
		}
	}
}

// IgnorePaths compiles the provided path patterns into regular expressions and adds them to the MiddlewareBuilder.
// This method allows specifying which paths the middleware should apply to.
// Parameters:
// - pathPatterns: a slice of strings representing the path patterns to be added.
// Returns:
// - the pointer to the MiddlewareBuilder instance to allow method chaining.
func (b *MiddlewareBuilder) IgnorePaths(pathPatterns ...string) *MiddlewareBuilder {
	// Initialize a slice to store the compiled regular expressions.
	//The capacity is set to the length of pathPatterns for efficiency.
	paths := make([]*regexp.Regexp, 0, len(pathPatterns))

	for _, pattern := range pathPatterns {
		// Attempt to compile the current pattern into a regular expression.
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			// If there's an error during compilation, log it and skip adding this pattern.
			log.Printf("failed to compile path pattern '%s': %v", pattern, err)
			continue
		}
		// Add the successfully compiled pattern to the slice of regular expressions.
		paths = append(paths, compiledPattern)
	}
	// Update the MiddlewareBuilder's Paths field with the compiled patterns.
	b.paths = paths
	return b // Return the MiddlewareBuilder instance for chaining.
}
