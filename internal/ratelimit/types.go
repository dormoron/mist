package ratelimit

import (
	"github.com/dormoron/mist"
)

// Limiter is an interface designed to abstract the rate-limiting logic.
// It confirms whether a particular action exceeds a predefined rate limit.
// Implementations of this interface can enforce rate limits in different ways,
// such as using a sliding window algorithm, token bucket, or any other strategy.
// Method: Limit
// The Limit method is responsible for determining if the given key has exceeded
// its allowable number of requests within a given time frame.
//
// Params:
//   - ctx: A pointer to a mist.Context object, which transports request-scoped values,
//     cancellation signals, deadlines, and other information across API boundaries.
//   - key: A string acting as the unique identifier to which the rate limit should be applied.
//
// Returns:
//   - A boolean indicating whether the request is within the rate limits (`true` if it's within limit, `false` if it exceeds).
//   - An error object that will be non-nil if an error occurs during the check (e.g., database or network issues).
type Limiter interface {
	Limit(ctx *mist.Context, key string) (bool, error)
}
