package redis

import (
	"context"
	"github.com/dormoron/mist/kit"
	"github.com/dormoron/mist/token"
	"github.com/redis/go-redis/v9"
	"time"
)

// Ensure Session implements the token.Session interface.
var _ token.Session = &Session{}

// Session object is a representation of the Redis-backed session store. Implements the token.Session interface.
type Session struct {
	client     redis.Cmdable // Redis client equipped with necessary commands.
	key        string        // Key to fetch the specific session in Redis.
	claims     token.Claims  // Claims associated with this session.
	expiration time.Duration // Time after which the session expires.
}

// Destroy method deletes the entire Session from Redis.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// Returns:
// - error: Potential error during session deletion.
func (sess *Session) Destroy(ctx context.Context) error {
	return sess.client.Del(ctx, sess.key).Err()
}

// Del method removes a specific field from the Session in Redis.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key of the field within the session data.
// Returns:
// - error: Potential error during field deletion.
func (sess *Session) Del(ctx context.Context, key string) error {
	return sess.client.Del(ctx, sess.key, key).Err()
}

// Set method saves a field with a specific value to the session in Redis.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key of the field in the session data.
// - val: The value to be saved in the session data.
// Returns:
// - error: Potential error during session field update.
func (sess *Session) Set(ctx context.Context, key string, val any) error {
	return sess.client.HSet(ctx, sess.key, key, val).Err()
}

// init method prepares the session and sets initial fields in Redis.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - kvs: A map of fields to be stored in the session.
// Returns:
// - error: Potential error during session initialization.
func (sess *Session) init(ctx context.Context, kvs map[string]any) error {
	pip := sess.client.Pipeline()
	for k, v := range kvs {
		pip.HMSet(ctx, sess.key, k, v)
	}
	pip.Expire(ctx, sess.key, sess.expiration)
	_, err := pip.Exec(ctx)
	return err
}

// Get method fetches a field from the session in Redis.
// Parameters:
// - ctx: The context carrying deadline and cancellation information.
// - key: The key of the field within the session.
// Returns:
// - kit.AnyValue: The value of the fetched field or an error if the fetching failed.
func (sess *Session) Get(ctx context.Context, key string) kit.AnyValue {
	res, err := sess.client.HGet(ctx, sess.key, key).Result()
	if err != nil {
		return kit.AnyValue{Err: err}
	}
	return kit.AnyValue{
		Val: res,
	}
}

// Claims method returns the claims associated with the session.
// Returns:
// - token.Claims: The set of claims.
func (sess *Session) Claims() token.Claims {
	return sess.claims
}

// initRedisSession method initializes a new session in Redis and returns a session object.
// Parameters:
// - ssid: The session ID string.
// - expiration: The duration after which the session should expire.
// - client: The Redis client used to perform commands.
// - cl: The set of claims associated with the session.
// Returns:
// - *Session: The initialized session object representing the Redis-based session.
func initRedisSession(
	ssid string,
	expiration time.Duration,
	client redis.Cmdable, cl token.Claims) *Session {
	return &Session{
		client:     client,            // Set the Redis client.
		key:        "session:" + ssid, // Create a key with specified session ID.
		expiration: expiration,        // Set the expiration time.
		claims:     cl,                // Attach claims to the Session.
	}
}
