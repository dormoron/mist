package ratelimit

import (
	"context"
)

// Limiter is an interface that defines a rate limiting method.
type Limiter interface {
	// Limit restricts the request rate based on the provided context and key.
	//
	// Parameters:
	//     ctx (context.Context): The context for the request, which is used to control the lifecycle of the request.
	//                            It can convey deadlines, cancellations signals, and other request-scoped values across API boundaries and goroutines.
	//     key (string): A unique string that identifies the request or resource to be limited.
	//                   This key is typically derived from the user's ID, IP address, or other identifying information.
	//
	// Returns:
	//     (bool, error): Returns a boolean indicating whether the request is allowed (true) or rate-limited (false).
	//                    If an error occurs during the process, it returns a non-nil error value.
	Limit(ctx context.Context, key string) (bool, error)
}
