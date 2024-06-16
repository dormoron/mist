package ratelimit

import (
	"context"
	_ "embed"
	"github.com/redis/go-redis/v9"
	"time"
)

// luaSlideWindow intentionally declared as global variable.
// It is a string variable which will contain the contents of 'slide_window.lua' after the file is embedded during the compile time.
// 'var' is used to declare a variable.
// 'luaSlideWindow' is the name of the variable. It's common in Go to use CamelCase for variable names
// 'string' is the type of the variable. This means the variable will hold a string data
// The type is inferred from the file contents by the //go:embed directive.
//
//go:embed slide_window.lua
var luaSlideWindow string

// RedisSlidingWindowLimiter struct is a structure in Go which represents a rate limiter using sliding window algorithm.
type RedisSlidingWindowLimiter struct {

	// Cmd is an interface from the go-redisess package (redis.Cmdable).
	// This interface includes methods for all Redis commands to execute queries.
	// Using an interface here instead of a specific type makes the limiter more flexible,
	// as it can accept any type that implements the `redis.Cmdable` interface, such as a Redis client or a Redis Cluster client.
	Cmd redis.Cmdable

	// Interval is of type time.Duration, representing the time window size for the rate limiter.
	// Interval is a type from the time package defining a duration or elapsed time in nanoseconds.
	// In terms of rate limiting, the interval is the time span during which a certain maximum number of requests can be made.
	Interval time.Duration

	// Rate is an integer that defines the maximum number of requests that can occur within the provided duration or interval.
	// For example, if Interval is 1 minute (`time.Minute`), and Rate is 100, this means a maximum of 100 requests can be made per minute.
	Rate int
}

// Limit is a method of the RedisSlidingWindowLimiter struct. It determines if a specific key has exceeded the allowed number of requests (rate) within the defined interval.
//
// Params:
//   - ctx: A context.Context object. It carries deadlines, cancellations signals, and other request-scoped values across API boundaries and between processes. It is often used for timeout management.
//   - key: A string that serves as a unique identifier for the request to be rate-limited.
//
// Returns:
//   - A boolean value indicating whether the request associated with the key is within the allowed rate limits. It returns `true` when the rate limit is not reached, and `false` otherwise.
//   - An error object that will hold an error (if any) that may have occurred during the function execution.
//
// The method uses the Eval command of the Redis server to execute a Lua script (luaSlideWindow) that implements the sliding window rate limit algorithm. It passes converted interval in milliseconds (r.Interval.Milliseconds()), maximum requests allowed (r.Rate), and the current Unix timestamp in milliseconds (time.Now().UnixMilli()) as parameters to the Lua script.
func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.Cmd.Eval(ctx, luaSlideWindow, []string{key}, r.Interval.Milliseconds(), r.Rate, time.Now().UnixMilli()).Bool()
}
