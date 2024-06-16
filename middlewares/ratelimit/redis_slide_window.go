package ratelimit

import (
	"github.com/dormoron/mist/internal/ratelimit"
	"github.com/redis/go-redis/v9"
	"time"
)

// InitRedisSlidingWindowLimiter initializes a rate limiter using a sliding window algorithm with Redis as the backend.
//
// This function is used to create an instance of a Redis-based sliding window rate limiter.
// The sliding window algorithm allows a more even distribution of requests over time,
// ensuring that the rate limit is not breached within the specified time interval.
//
// Parameters:
//
//	cmd (redisess.Cmdable): The Redis client or connection that supports the required Redis commands.
//	                     This can be any type that implements the redisess.Cmdable interface, such as
//	                     *redisess.Client or *redisess.ClusterClient.
//	interval (time.Duration): The time duration representing the size of the sliding window.
//	                          This determines the period over which the rate is calculated.
//	rate (int): The maximum number of requests allowed within the specified interval.
//
// Returns:
//
//	(ratelimit.Limiter): An implementation of the rate limiting interface using the sliding window algorithm,
//	                     backed by Redis. This object can be used to apply rate limiting logic to various operations.
func InitRedisSlidingWindowLimiter(
	cmd redis.Cmdable,
	interval time.Duration,
	rate int,
) ratelimit.Limiter {
	return &ratelimit.RedisSlidingWindowLimiter{
		Cmd:      cmd,
		Interval: interval,
		Rate:     rate,
	}
}
