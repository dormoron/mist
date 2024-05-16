package token

import (
	"github.com/dormoron/mist"
	"log/slog"
	"net/http"
)

// MiddlewareBuilder is a type that encapsulates session management within middleware.
//
// Fields:
// - sp: An instance of a Provider that manages session-related operations.
type MiddlewareBuilder struct {
	sp Provider
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
