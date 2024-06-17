package redisess

import (
	"context"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
	"github.com/redis/go-redis/v9"
	"time"
)

// Ensure that Session implements the security.Session interface.
var _ security.Session = &Session{}

// Session is a struct that manages session data using Redis as the backend storage.
type Session struct {
	client     redis.Cmdable   // Redis client used to interact with the Redis server.
	key        string          // The Redis key under which the session data is stored.
	claims     security.Claims // Security claims associated with the session.
	expiration time.Duration   // The expiration duration of the session.
}

// Destroy deletes the session data from Redis.
// Parameters:
// - ctx: The context for controlling the request lifetime (context.Context).
// Returns:
// - error: An error if the deletion fails.
func (sess *Session) Destroy(ctx context.Context) error {
	return sess.client.Del(ctx, sess.key).Err() // Perform a DEL command on the session key.
}

// Del deletes a specific key from the session data in Redis.
// Parameters:
// - ctx: The context for controlling the request lifetime (context.Context).
// - key: The specific key to be deleted from the session data (string).
// Returns:
// - error: An error if the deletion fails.
func (sess *Session) Del(ctx context.Context, key string) error {
	return sess.client.Del(ctx, sess.key, key).Err() // Perform a DEL command on the session key and specific field.
}

// Set sets a key-value pair in the session data in Redis.
// Parameters:
// - ctx: The context for controlling the request lifetime (context.Context).
// - key: The key to be set in the session data (string).
// - val: The value to be set for the specified key (any).
// Returns:
// - error: An error if the set operation fails.
func (sess *Session) Set(ctx context.Context, key string, val any) error {
	return sess.client.HSet(ctx, sess.key, key, val).Err() // Perform an HSET command to set the key-value pair in the hash.
}

// init initializes the session data with key-value pairs provided in the map kvs.
// Parameters:
// - ctx: The context for controlling the request lifetime (context.Context).
// - kvs: A map of key-value pairs to initialize the session data (map[string]any).
// Returns:
// - error: An error if the initialization fails.
func (sess *Session) init(ctx context.Context, kvs map[string]any) error {
	pip := sess.client.Pipeline() // Create a new pipeline to batch the commands.
	for k, v := range kvs {
		pip.HMSet(ctx, sess.key, k, v) // Add an HMSET command for each key-value pair.
	}
	pip.Expire(ctx, sess.key, sess.expiration) // Set the expiration time for the session.
	_, err := pip.Exec(ctx)                    // Execute all the commands in the pipeline.
	return err                                 // Return any error that occurred during execution.
}

// Get retrieves the value associated with a specific key from the session data in Redis.
// Parameters:
// - ctx: The context for controlling the request lifetime (context.Context).
// - key: The key to retrieve from the session data (string).
// Returns:
// - mist.AnyValue: The value associated with the key, or an error if the retrieval fails.
func (sess *Session) Get(ctx context.Context, key string) mist.AnyValue {
	res, err := sess.client.HGet(ctx, sess.key, key).Result() // Perform an HGET command to retrieve the value.
	if err != nil {
		return mist.AnyValue{Err: err} // Return an AnyValue with the error if the retrieval fails.
	}
	return mist.AnyValue{
		Val: res, // Return an AnyValue with the retrieved value.
	}
}

// Claims returns the security claims associated with the session.
// Returns:
// - security.Claims: The claims associated with the session.
func (sess *Session) Claims() security.Claims {
	return sess.claims // Return the claims.
}

// initRedisSession initializes a new Session instance with the given parameters.
// Parameters:
// - ssid: The session ID (string).
// - expiration: The expiration duration of the session (time.Duration).
// - client: The Redis client used to interact with the Redis server (redis.Cmdable).
// - cl: Security claims associated with the session (security.Claims).
// Returns:
// - *Session: A pointer to the newly created Session instance.
func initRedisSession(ssid string, expiration time.Duration, client redis.Cmdable, cl security.Claims) *Session {
	return &Session{
		client:     client,            // Set the Redis client.
		key:        "session:" + ssid, // Construct the Redis key using the session ID.
		expiration: expiration,        // Set the expiration duration.
		claims:     cl,                // Set the security claims.
	}
}
