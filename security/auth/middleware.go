package auth

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/auth/kit"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

// MiddlewareBuilder is a generic struct for constructing middleware for type T.
type MiddlewareBuilder[T any] struct {
	// ignorePath is a function that determines if a given path should be ignored by middleware.
	ignorePath func(path string) bool

	// manager is a pointer to an instance of Management which handles token management.
	manager *Management[T]

	// nowFunc is a function that returns the current time, used for token validation.
	nowFunc func() time.Time
}

// initMiddlewareBuilder initializes a new MiddlewareBuilder instance with the provided Management instance.
// Parameters:
// - m: A pointer to the Management instance to use ('*Management[T]').
// Returns:
// - *MiddlewareBuilder[T]: A pointer to the initialized MiddlewareBuilder instance.
func initMiddlewareBuilder[T any](m *Management[T]) *MiddlewareBuilder[T] {
	return &MiddlewareBuilder[T]{
		manager: m,
		ignorePath: func(path string) bool {
			return false // By default, don't ignore any path.
		},
		nowFunc: m.nowFunc, // Use the nowFunc from the provided Management instance.
	}
}

// IgnorePath sets the paths that should be ignored by middleware. This method internally calls IgnorePathFunc.
// Parameters:
// - path: Variadic list of paths to ignore ('...string').
// Returns:
// - *MiddlewareBuilder[T]: The MiddlewareBuilder instance, to allow for method chaining.
func (m *MiddlewareBuilder[T]) IgnorePath(path ...string) *MiddlewareBuilder[T] {
	return m.IgnorePathFunc(staticIgnorePaths(path...))
}

// IgnorePathFunc sets a custom function that determines if a given path should be ignored by the middleware.
// Parameters:
// - fn: Function that determines if a path should be ignored ('func(path string) bool').
// Returns:
// - *MiddlewareBuilder[T]: The MiddlewareBuilder instance, to allow for method chaining.
func (m *MiddlewareBuilder[T]) IgnorePathFunc(fn func(path string) bool) *MiddlewareBuilder[T] {
	m.ignorePath = fn
	return m
}

// Build constructs the middleware using the settings configured in the MiddlewareBuilder instance.
// Returns:
// - mist.Middleware: A middleware function that processes the request.
func (m *MiddlewareBuilder[T]) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// Check if the request path should be ignored.
			if m.ignorePath(ctx.Request.URL.Path) {
				next(ctx)
				return
			}

			// Extract the token from the request.
			tokenStr := m.manager.extractTokenString(ctx)
			if tokenStr == "" {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			// Verify the access token.
			clm, err := m.manager.VerifyAccessToken(tokenStr, jwt.WithTimeFunc(m.nowFunc))
			if err != nil {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			// Set the claims in the context.
			m.manager.SetClaims(ctx, clm)
			next(ctx)
		}
	}
}

// staticIgnorePaths creates a function that checks if a given path is in the list of paths to ignore.
// Parameters:
// - paths: Variadic list of paths to ignore ('...string').
// Returns:
// - func(path string) bool: A function that returns true if the path is in the ignore list.
func staticIgnorePaths(paths ...string) func(path string) bool {
	s := kit.NewMapSet[string](len(paths))
	for _, path := range paths {
		s.Add(path) // Add each path to the set.
	}
	return func(path string) bool {
		return s.Exist(path) // Check if the path exists in the set.
	}
}
