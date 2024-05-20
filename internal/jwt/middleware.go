package jwt

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/kit"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"time"
)

// MiddlewareBuilder provides templates for constructing middleware relevant to authentication.
//
// Fields:
// - ignorePath: A function used to determine if the provided path should be ignored by the middleware.
// - manager: A pointer to Management the provides tools to manage JWT tokens and their lifecycle.
// - nowFunc: A function that returns the current time, used for token expiry checks.
//
// Generics:
// - T: A type parameter that allows the builder to be used with various data types.
type MiddlewareBuilder[T any] struct {
	ignorePath func(path string) bool
	manager    *Management[T]
	nowFunc    func() time.Time
}

// initMiddlewareBuilder initializes and returns a new instance of MiddlewareBuilder.
//
// Parameters:
// - m: A pointer to a Management instance that handles token operations.
//
// Returns:
// - *MiddlewareBuilder[T]: A pointer to a new MiddlewareBuilder instance.
func initMiddlewareBuilder[T any](m *Management[T]) *MiddlewareBuilder[T] {
	return &MiddlewareBuilder[T]{
		manager: m,
		// ignorePath is a default function that always returns false, meaning no path is ignored initially.
		ignorePath: func(path string) bool {
			return false
		},
		// nowFunc is assigned from the Management instance to ensure consistent time handling.
		nowFunc: m.nowFunc,
	}
}

// IgnorePath sets the paths that should be ignored by the middleware and returns the MiddlewareBuilder.
// Any requests matching the ignored paths will skip token validation.
//
// Parameters:
// - path: A list of strings that represent the paths to ignore.
//
// Returns:
// - *MiddlewareBuilder[T]: A pointer to the MiddlewareBuilder for method chaining.
func (m *MiddlewareBuilder[T]) IgnorePath(path ...string) *MiddlewareBuilder[T] {
	return m.IgnorePathFunc(staticIgnorePaths(path...))
}

// IgnorePathFunc sets a custom function to determine if middleware should ignore a path.
//
// Parameters:
// - fn: A function that takes a path string as input and returns a bool indicating if the path should be ignored.
//
// Returns:
// - *MiddlewareBuilder[T]: A pointer to the MiddlewareBuilder for method chaining.
func (m *MiddlewareBuilder[T]) IgnorePathFunc(fn func(path string) bool) *MiddlewareBuilder[T] {
	m.ignorePath = fn
	return m
}

// Build constructs the middleware function that can be integrated into an HTTP handling pipeline.
//
// Returns:
// - mist.Middleware: The middleware with embedded logic for token validation and path ignoring.
func (m *MiddlewareBuilder[T]) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// If the current path should be ignored, the middleware allows the request to proceed without token validation.
			if m.ignorePath(ctx.Request.URL.Path) {
				return
			}
			// Attempt to extract the token from the context using the Management instance.
			tokenStr := m.manager.extractTokenString(ctx)
			if tokenStr == "" {
				slog.Debug("failed to extract token")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			// Verify the token and get the claims. If there's an error, abort with an HTTP 401 Unauthorized status.
			clm, err := m.manager.VerifyAccessToken(tokenStr,
				jwt.WithTimeFunc(m.nowFunc))
			if err != nil {
				slog.Debug("access token verification failed")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			// If token validation is successful, set the claims in the context for the downstream handlers.
			m.manager.SetClaims(ctx, clm)
			// Call the next middleware or final handler in the chain.
			next(ctx)
		}
	}
}

// staticIgnorePaths helps to create a function that ignores specific paths. It initializes a set with provided paths.
//
// Parameters:
// - paths: A variadic number of strings representing the paths to be checked against.
//
// Returns:
// - func(path string) bool: A function that takes a path string as an argument and returns true if the path is part of the ignored list.
func staticIgnorePaths(paths ...string) func(path string) bool {
	// Initialize a set with the capacity based on the number of paths provided.
	s := kit.InitMapSet[string](len(paths))
	// Add each provided path to the set.
	for _, path := range paths {
		s.Add(path)
	}
	// Return a function that checks if a given path is contained in the initialized set.
	return func(path string) bool {
		return s.Exist(path)
	}
}
