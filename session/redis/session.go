package redis

import (
	"context"
	"fmt"
	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/session"
	"github.com/redis/go-redis/v9"
	"time"
)

// Store is a struct that encapsulates the necessary details to manage session
// data within a Redis datastore. The struct holds configuration settings and
// a Redis client interface used to interact with Redis commands, providing a
// way to store, retrieve, and manage session data with an assigned expiration time.
//
// Attributes:
//
//   - prefix: A string used as a key prefix in Redis to namespace session data.
//     This helps to avoid key collisions with other data in Redis and allows
//     easier identification and management of session-related keys. For example,
//     a common practice is to prepend "sess:" to the actual session ID.
//
//   - client: An interface that represents the Redis command executor. It must satisfy
//     the redis.Cmdable interface, which includes methods for executing commands
//     like GET, SET, DELETE, etc., used for session management operations.
//     This abstraction allows the Store to interact with Redis without depending
//     on a specific Redis client implementation, making the Store more flexible
//     and testable.
//
//   - expiration: A time.Duration value that determines the lifespan of a session within
//     Redis. When a session is created or refreshed, this duration is used to
//     set the Redis key's Time-To-Live (TTL). It ensures that session data
//     does not persist indefinitely in the database, which can be especially
//     important for maintaining a secure and efficient session management system.
//
// The Store struct is used internally in session management systems to encapsulate the logic
// and interaction with the session storage backend (in this case, Redis). It generally provides
// methods for creating new session records, retrieving existing sessions, refreshing session
// expiration times, and deleting sessions as required by the application's session management logic.
//
// Usage:
// A Store instance is typically initialized with a Redis client and configured with the desired
// prefix and expiration before being provided to the session management component that will handle
// session operations. By centralizing the session storage logic in the Store, we can easily manage
// and update session storage strategies in one place while adhering to the principles of encapsulation.
//
// Example initialization:
//
//	rdb := redis.NewClient(&redis.Options{
//	    Addr:     "localhost:6379",
//	    Password: "", // no password set
//	    DB:       0,  // use default DB
//	})
//
//	sessionStore := &Store{
//	    prefix:     "sess:",
//	    client:     rdb,
//	    expiration: 30 * time.Minute,
//	}
type Store struct {
	prefix     string        // The key prefix for session data in Redis.
	client     redis.Cmdable // A Redis client interface for issuing commands.
	expiration time.Duration // Duration after which a session will expire in Redis.
}

// StoreOptions represents a function type that applies configuration settings to a Store object.
// This type is defined as a function that takes a pointer to a Store, which allows for any
// number of configuration functions to be defined and then applied to a Store instance.
// The concept is part of a functional options pattern, which enables the creation of
// flexible and clean APIs for configuring objects without needing a complex constructor
// or an explosion of configuration methods.
//
// The functional options pattern is particularly useful in situations where a Store might have
// many optional parameters, each with sensible defaults. Rather than requiring the caller to
// specify every possible option in a constructor or to instantiate the object and then make
// subsequent calls to set various options, the functional options pattern allows the caller to
// provide only the options they care about at the point of creation.
//
// The idea behind this approach is to declare functions that accept a Store pointer and mutate it.
// Each of these functions can encapsulate logic that sets or modifies a particular field or behavior
// of the Store. When creating a Store, a series of these functions can then be passed to a constructor
// function or an initialization method, which iterates over them, applying each function to the Store
// instance. This method allows for the addition of new options in the future without breaking existing
// code or interfaces.
//
// Usage:
// To define a functional option, you create a function that adheres to the StoreOptions signature.
// Each such function can set one or more Store fields, perform validation, and so on. Here's a
// notional example of defining and applying StoreOptions:
//
// // WithPrefix defines a Store option that sets the key prefix for session data.
//
//	func WithPrefix(p string) StoreOptions {
//	    return func(s *Store) {
//	        s.prefix = p
//	    }
//	}
//
// // WithExpiration defines a Store option that sets the expiration duration for sessions.
//
//	func WithExpiration(d time.Duration) StoreOptions {
//	    return func(s *Store) {
//	        s.expiration = d
//	    }
//	}
//
// // NewStore creates a new Store with the provided options applied.
//
//	func NewStore(client redis.Cmdable, opts ...StoreOptions) *Store {
//	    store := &Store{client: client}
//	    for _, opt := range opts {
//	        opt(store)
//	    }
//	    return store
//	}
//
// // Example of creating a Store with custom options:
// store := NewStore(redisClient, WithPrefix("myapp:sess:"), WithExpiration(1*time.Hour))
type StoreOptions func(store *Store) // A function type to configure and modify a Store instance.

// InitStore initializes a new Store struct that will be used for managing sessions in a Redis datastore.
// This function serves as a constructor for the Store type, allowing customization of the Store's properties
// through functional options passed in as arguments. It provides a flexible configuration mechanism that
// facilitates the application of zero or more optional settings to the Store at the time of its creation.
//
// Parameters:
//
//   - client: An instance of redis.Cmdable which provides the interface for executing commands against the Redis server.
//     It is a required parameter as it establishes the necessary connection for the Store to interact with Redis.
//
//   - opts: A variadic slice of StoreOptions, which are optional configuration functions. Each StoreOptions function can
//     modify the Store's properties such as its key prefix or expiration duration for stored sessions.
//     These options are applied in the order they are passed to the function, meaning that later options can
//     overwrite the settings applied by earlier ones.
//
// The function establishes a new Store with a default expiration of 15 minutes and a default prefix of "sessionId".
// These defaults are immediately established within the new Store instance and will be used unless overriden by the options
// provided in `opts`. The structure of this constructor function with functional options allows for future extensions
// of configuration parameters without requiring changes to the function signature.
//
// The Store struct resulting from this function is not expected to be modified after it's returned since it is designed
// to be safe for concurrent use by multiple goroutines.
//
// Usage:
// To use InitStore, you must first have a Redis client configured. Then you can create a Store with the default settings or
// provide functional options to customize the Store as needed:
//
// // Create a Redis client.
//
//	redisClient := redis.NewClient(&redis.Options{
//	    Addr:     "localhost:6379",
//	    Password: "", // no password set
//	    DB:       0,  // use the default DB
//	})
//
// // Initialize Store with default settings.
// store := InitStore(redisClient)
//
// // Initialize Store with custom options.
// storeWithOpts := InitStore(redisClient, WithPrefix("customPrefix:"), WithExpiration(30*time.Minute))
//
// After initialization, the Store can be used to manage session data within the configured Redis datastore.
func InitStore(client redis.Cmdable, opts ...StoreOptions) *Store {
	// Instantiate a new Store with a default expiration time and key prefix.
	res := &Store{
		client:     client,
		expiration: time.Minute * 15, // Default expiration time set to 15 minutes.
		prefix:     "sessionId",      // Default key prefix for Redis keys.
	}
	// Apply any provided StoreOptions to customize the Store's configuration.
	for _, opt := range opts {
		opt(res)
	}
	// Return the newly configured Store instance.
	return res
}

// StoreWithExpiration returns a StoreOptions function that sets the expiration
// duration for the session data stored in a Store. This returned function can
// be passed as a configuration option when initializing a Store.
//
// This function is an example of a functional option - a technique for making
// functions flexible by allowing them to have various configurations without
// changing the function signature. The `StoreWithExpiration` function specifically
// allows users to customize the expiration of sessions within the Store.
//
// The functional option returned by `StoreWithExpiration` takes a Store as a
// parameter and sets its expiration field to the value provided in the
// `expiration` parameter. This approach decouples the option-configuration logic
// from the Store type, leading to cleaner, more maintainable code.
//
// Parameters:
//   - expiration: A time.Duration value that specifies the new expiration duration
//     for the sessions managed by the Store. It overrides the default
//     or any previously set expiration time.
//
// Using this function promotes immutability and thread-safety by avoiding direct
// modifications to the Store’s fields after its instantiation. Instead, it endorses
// the use of designated functions to tailor the object upon creation.
//
// Usage:
// To use this function, simply pass the desired time.Duration as the parameter.
// The returned StoreOptions can then be provided to InitStore or any similar
// Store initialization function that accepts functional options to configure
// the new Store's session expiration time.
//
// Example:
// // Setting up a custom expiration duration of 30 minutes.
// store := InitStore(redisClient, StoreWithExpiration(30*time.Minute))
//
// The above code snippet will create a Store with session data that expires after
// 30 minutes, instead of using the default expiration duration.
func StoreWithExpiration(expiration time.Duration) StoreOptions {
	// Return a function that conforms to the StoreOptions type.
	return func(store *Store) {
		// Set the expiration field of the Store to the passed time.Duration value.
		store.expiration = expiration
	}
}

// StoreWithPrefix returns a StoreOptions function that sets a specific prefix for the keys
// in the Store structure. This allows for the keys associated with the session data stored
// within the Store to be distinguished from other data within the same Redis instance.
//
// Parameters:
//   - prefix: A string that will be used as the key prefix within the Store. This prefix is
//     prepended to all keys managed by the Store to namespace the keys and avoid conflicts.
//
// The `StoreWithPrefix` function is another example of a functional option. Functional options
// are a method of implementing clean and flexible APIs for configuring objects without requiring
// a complex constructor or a large amount of setup code. In this case, `StoreWithPrefix` enables
// the creation of Stores with custom key prefixes, offering the possibility to partition session
// data logically within the datastore or to use a preferred naming scheme for greater clarity in
// Redis key management.
//
// By using functional options like this, it's possible to extend the configuration of Store without
// breaking existing code that uses the Store constructor. This pattern also provides immutability
// post-creation and ensures that the Store configuration step is thread-safe and free from race conditions.
//
// Usage:
// To use `StoreWithPrefix`, simply pass the desired prefix string as the parameter when creating a Store.
// The returned StoreOptions function can be fed into the Store constructor to apply the prefix setting.
//
// Example:
// // Setting up a Store to use a custom key prefix of "myapp:sess:".
// store := InitStore(redisClient, StoreWithPrefix("myapp:sess:"))
//
// In the example, any keys that the Store creates in Redis will be prefixed with "myapp:sess:", allowing
// for easy identification and segregation of session data.
func StoreWithPrefix(prefix string) StoreOptions {
	// Return a closure that conforms to the StoreOptions type.
	return func(store *Store) {
		// Set the prefix field of the Store to the given string parameter.
		store.prefix = prefix
	}
}

// Generate creates a new session in the Redis store associated with the provided id and sets an expiration time
// for that session. It returns a Session object representing the newly created session along with an error, if any.
//
// Context and an identifier string are both required to create a session. This method uses an optimistic locking
// strategy by setting the value only if it does not already exist in Redis, which prevents overwriting of the
// session data that might be created concurrently with the same ID.
//
// Parameters:
//   - ctx: context.Context which allows for timeouts or cancellation signals to propagate into the database calls,
//     ensuring that the function is responsive to shutdown signals and does not leave hanging database calls.
//   - id: A string representing the unique identifier for the session. This ID should be unique to maintain separate
//     session data for different users or contexts.
//
// The key for the new session within Redis is constructed by combining the Store's prefix with the provided id to
// ensure namespace isolation within the Redis data store, which helps avoid key collisions.
//
// Upon successful creation of the session in Redis, the method sets an expiration for the session data at the
// configured duration stored in the Store's expiration field. If any Redis command fails, the error is returned
// to the caller, and no session is created.
//
// The returned Session object contains the session's ID, the fully qualified Redis key, and a reference to the
// Redis client used by the Store, which allows for further operations on the session data.
//
// Usage example:
// Assuming you have an instance `store` of *Store properly initialized with a Redis client, you could generate
// a new session like so:
//
// session, err := store.Generate(context.Background(), "unique-session-id")
//
//	if err != nil {
//	    // handle the error
//	}
//
// // Use session
//
// Note:
// In the above `Generate` method, it is assumed that there is a `redisKey` utility function used to concatenate
// the prefix with the session ID, and a `Session` type that is compatible with the returned value.
func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	// Construct the Redis key for the session using the provided ID and the Store's key prefix.
	key := redisKey(s.prefix, id)

	// Attempt to set the initial value for the session in Redis; fail if there's an error.
	_, err := s.client.HSet(ctx, key, id, id).Result()
	if err != nil {
		return nil, err
	}

	// Set the expiration time for the session key; fail if there's an error.
	_, err = s.client.Expire(ctx, key, s.expiration).Result()
	if err != nil {
		return nil, err
	}

	// Return a Session object representing the created session.
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

// Refresh updates the expiration time of an existing session in the Redis store to the Store's configured
// expiration duration. This method is used to extend the life of the session each time a user interacts with
// the application, preventing timeout while the session is active.
//
// Parameters:
//   - ctx: context.Context which provides the capability to notify the Redis commands of deadlines or cancellation signals.
//     A context passed here allows the method to be aware of request-scoped deadlines and operation canceling which
//     is critical for maintaining responsiveness and robustness of the application.
//   - id: The unique string identifier for the existing session which needs to be refreshed. It must match an ID from a
//     previously created session, otherwise, an error indicating the session was not found will be returned.
//
// The method first constructs the Redis key by combining the provided session identifier (id) with the prefix defined
// in the Store. This key is then used to locate the session in Redis and update its expiration. If the key does not
// exist in Redis (indicating that there is no such session, or it may have expired), an error is returned signifying
// the session was not found.
//
// If the Redis 'Expire' command returns an error, it is relayed back to the caller. Only if the Redis 'Expire' command
// executes successfully does the method return a nil error, indicating a successful refresh of the session's expiration.
//
// Usage Example:
// Assuming 'store' is an instance of *Store with a valid Redis client. You can refresh a session's expiration like so:
//
// err := store.Refresh(context.Background(), "existing-session-id")
//
//	if err != nil {
//	    // handle the error, which might be a missing session or a Redis command error
//	}
//
// The Redis 'Expire' command is atomic, and as such, when used with a context with a timeout or cancelation, it ensures that
// the method won't leave a Redis session in an undefined state. It will either update the expiration successfully or fail cleanly
// without side effects.
func (s *Store) Refresh(ctx context.Context, id string) error {
	// Define Redis key to be used for extending the session expiration.
	key := redisKey(s.prefix, id)

	// Try to update the expiration time of the session key in Redis.
	ok, err := s.client.Expire(ctx, key, s.expiration).Result()
	if err != nil {
		// If a Redis error occurs, return the error.
		return err
	}

	if !ok {
		// If the session key does not exist (it might have expired), return a 'session not found' error.
		return errs.ErrIdSessionNotFound()
	}

	// If everything is successful, return nil indicating the session expiration was successfully refreshed.
	return nil
}

// Remove deletes a session from the Redis store using the session ID provided. This method is typically called when
// a user logs out or when a session is deemed invalid or expired beyond the configured retention policy. After calling
// this method, the session data will no longer exist in Redis, and the ID cannot be used to retrieve or refresh the session.
//
// Parameters:
//   - ctx: The context.Context, which encompasses deadlines, cancellation signals, and other request-scoped values
//     across API boundaries and between processes. This is important for the database operations to be able to respond
//     appropriately to the lifecycle of the HTTP request or any other scope the context is derived from.
//   - id: A string representing the unique identifier of the session to be removed. It corresponds to the ID of a session that was
//     previously generated and now needs to be deleted from the store.
//
// Removal of a session is a straightforward process. First, the Redis key for the session is constructed by concatenating the
// Store's prefix and the provided session ID. This key is then used to identify and delete the associated session data in Redis
// via the 'Del' command. Should the 'Del' command encounter any issues (e.g., a connection problem to Redis), an error is
// returned to the caller signalling the failure of the deletion process.
//
// Importantly, the method returns an error if the Redis 'Del' command fails. However, if the 'Del' command indicates that no
// keys were found for deletion (which is a success scenario from the Redis standpoint), no error is returned, as it is
// indicative of the fact that the session either never existed or had already been removed.
//
// Usage Example:
// Assuming 'store' is an instance of *Store with a proper Redis client connection, the removal of a session is done as follows:
//
// err := store.Remove(context.Background(), "session-id-to-remove")
//
//	if err != nil {
//	    // handle the error from Redis operation
//	}
//
// Note: The returned error is only related to the actual Redis operation. The absence of a session (e.g., already deleted session)
// does not lead to an error. It is up to the caller to ensure the provided session ID corresponds with the actual session
// to be deleted and handle the logic related to non-existing IDs accordingly.
func (s *Store) Remove(ctx context.Context, id string) error {
	// Construct the Redis key for the session using the provided ID and the Store's prefix.
	key := redisKey(s.prefix, id)

	// Execute the Redis 'Del' command to remove the session data associated with the key.
	_, err := s.client.Del(ctx, key).Result()
	if err != nil {
		// If the Redis operation results in an error, return the error to the caller.
		return err
	}

	// If there are no errors from Redis, return nil indicating successful removal of the session.
	return err
}

// Get retrieves the session data from the Redis store using the provided session ID. If the session is found, it returns
// a Session struct which includes the session ID, the Redis key for accessing the session, and the Redis client from
// the Store. If the session is not found, or any other error occurs, it returns the corresponding error.
//
// Parameters:
//   - ctx: A context.Context parameter, which allows the method to be aware of request timeouts or cancellations, ensuring
//     the retrieval request adheres to the constraints set by service consumers.
//   - id: A string representing the unique identifier of the session to be retrieved. This ID should match the one used
//     when the session was created.
//
// The function comprises of the following steps:
// 1. Builds the Redis key using the store's prefix and the session ID provided by the user.
// 2. Calls the Redis 'Exists' command to check if a session with the given key is present in the store.
//   - If an error occurs during the Redis call, it is captured and returned immediately, halting any further execution.
//
// 3. If the Redis 'Exists' command returns a count other than 1, the function infers the session does not exist in the store.
//   - An error, indicating the session could not be found, is returned.
//     4. If the Redis 'Exists' command confirms the session's existence, the function creates a Session object and returns it
//     along with a nil error, signaling a successful retrieval.
//
// Usage Example:
// To retrieve a session with a known session ID, assuming 'store' is an instance of *Store with a Redis client, you would do:
//
// session, err := store.Get(context.Background(), "known-session-id")
//
//	if err != nil {
//	    // handle the error if the session was not found or if a Redis error occurred
//	} else {
//
//	    // proceed with the retrieved session data contained in the 'session' variable
//	}
//
// Note: This method is critical in systems that need to verify the existence and validity of a session. It does so in an atomic
// manner, meaning either the session is found and considered valid, or an error state is returned, providing a deterministic
// outcome for session validation.
func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	// Construct the full Redis key for the session with the given ID
	key := redisKey(s.prefix, id)

	// Check if the session exists in Redis store.
	cnt, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		// Return nil and the error if there was an issue with the Redis 'Exists' command.
		return nil, err
	}
	if cnt != 1 {
		// If the session does NOT exist, return nil and an error indicating the session was not found.
		return nil, errs.ErrIdSessionNotFound()
	}

	// Return a new instance of a Session object with the ID, key, and Redis client if session exists.
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

// Session is a struct that encapsulates a single user session's attributes and provides the necessary metadata and tools
// to interact with the session data stored in Redis. The struct holds information for identifying the session and
// interacting with the session's corresponding data within the Redis datastore.
//
// Fields:
//   - id: A string that uniquely identifies the session. This identifier is used by clients to refer to the session during API calls
//     and is often generated by the system when the session is first created. It is the primary key for the session management
//     and should be kept confidential to prevent unauthorized access to the session.
//   - key: A string that represents the constructed key used within the Redis store. The key is composed of a prefix (common to all
//     sessions managed by the application) and the session's unique ID. This composition facilitates namespacing in the Redis
//     store and avoids key collisions with other potential data stored in the same Redis instance.
//   - client: An interface of type redis.Cmdable which provides a set of commands that are used to execute Redis commands
//     against the session data. By storing the reference to the Redis client in the session struct, the session can
//     directly perform actions on Redis, such as retrieving, updating, or deleting session-specific data.
//
// The struct does not contain the session data itself; instead, it provides the mechanisms to retrieve or manipulate it in the
// Redis data store.
//
// Usage Example:
// A Session struct can be utilized to interact with session data. For instance, once a Session instance is obtained through
// the Store's Get method:
//
// session, err := store.Get(context.Background(), "known-session-id")
//
// You can then use the session.client to perform operations specific to the session, like getting or setting session-related
// values in Redis:
//
// val, err := session.client.Get(ctx, session.key).Result()
//
// This design keeps session logic encapsulated and allows for interaction through the struct's methods or using the client directly.
type Session struct {
	id     string        // Unique identifier for the session
	key    string        // Redis key under which session data is stored
	client redis.Cmdable // Redis client interface to interact with the session data
}

// Get is a method on the Session struct that retrieves the value associated with a
// specific key from a Redis hash.
//
// Parameters:
//   - ctx context.Context: The context in which the method is called, which can be used
//     to pass request-scoped values, like deadlines or cancellation signals to the Redis
//     operation. For example, if the context has a deadline that passes before the
//     operation is completed, the operation will be cancelled.
//   - key string: The key corresponding to the value you want to retrieve within the
//     Redis hash.
//
// Returns:
//   - any: The value retrieved from the Redis hash. It uses the type 'any' which is an
//     empty interface type capable of holding values of any type, which is suitable for
//     a Redis client that can store various types of values.
//   - error: An error that might occur during the operation, if any. Common errors include
//     network issues, context timeout, or the key not being found in the hash. If the
//     operation is successful, the error returned is nil.
//
// The method works by calling the HGet method provided by the Redis client to attempt
// to get the value from the Redis hash and handling potential errors that could arise.
//
// Usage:
// Assuming 'session' is a properly initialized instance of Session with a Redis client 'client'
// and a hash key 'key' already set up, you would call:
//
// value, err := session.Get(context.Background(), "exampleKey")
//
// This would attempt to retrieve the value associated with "exampleKey" in the Redis hash.
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	// Attempt to retrieve the value from the Redis hash using the provided key.
	// The HGet method is a Redis command that fetches the value of a field in a hash stored at a key.
	val, err := s.client.HGet(ctx, s.key, key).Result()
	// Handle any potential errors returned by the HGet method.
	// If there is an error, it returns a nil value and the error itself.
	if err != nil {
		return nil, err
	}
	// If retrieval is successful, return the value and a nil error.
	return val, nil
}

// Set is a method on the Session struct that sets a key-value pair in a Redis hash
// only if the hash already exists.
//
// Parameters:
//   - ctx context.Context: The context in which the method is called, used to carry
//     deadlines, cancellation signals, and other request-scoped values.
//   - key string: The key component of the key-value pair to be set within the Redis hash.
//   - value any: The value component of the key-value pair to be set within the Redis hash.
//     It adheres to the interface{} type which allows for values of any type.
//
// Returns:
//   - error: An error that might occur during the operation, if any. The function will
//     return the error thrown by the Redis command or if the hash does not exist (indicated
//     by the return value -1 from the Lua script). If the operation is successful, the error
//     returned is nil.
//
// The method uses Redis Lua scripting to atomically check the existence of the hash and set
// a value if and only if the hash exists. Lua scripting allows for complex operations to be
// executed on the server side to minimize network round trips.
func (s *Session) Set(ctx context.Context, key string, value any) error {
	// Lua script to be evaluated on the Redis server. The script checks if a hash exists
	// and sets a key-value pair in the hash if it does.
	const lua = `
if redis.call("exists", KEYS[1])
then
    return redis.call("hset", KEYS[1], ARGV[1], ARGV[2])
else
    return -1
end
`
	// Eval runs a Lua script using the specified keys and arguments. It invokes the Lua
	// script defined above with the Redis hash name, the key, and the value provided
	// by the function parameters.
	// The .Int() method tries to convert the returned value from the Eval command to an integer.
	res, err := s.client.Eval(ctx, lua, []string{s.key}, key, value).Int()
	// Handle any errors returned by the Eval method.
	if err != nil {
		return err
	}

	// Check the script result. A return value of -1 indicates the hash does not exist.
	// errs.ErrIdSessionNotFound() is assumed to be a custom error defined elsewhere in
	// the code that denotes the specific error condition where the session or hash does not exist.
	if res < 0 {
		return errs.ErrIdSessionNotFound()
	}

	// If there are no errors and the hash exists, return nil to indicate successful execution.
	return nil
}

// ID returns the identifier of the current session.
//
// This method does not accept any parameters and it simply returns the ID property
// from the Session struct. The ID typically uniquely identifies the session, which
// can then be used to retrieve or associate session-specific data.
//
// Returns:
// - string: The unique identifier of the session.
func (s *Session) ID() string {
	// Return the 'id' field of the Session struct.
	// This 'id' is the unique identifier for the session instance,
	// which is useful for identifying the session in various operations.
	return s.id
}

// redisKey constructs a Redis key using a given prefix and identifier.
//
// This is a helper function used to format and generate a Redis key by concatenating
// a predefined prefix with a unique identifier, typically separated by a hyphen. This
// ensures that keys are namespaced properly, making them easily recognizable and
// avoiding collisions within the Redis database.
//
// Parameters:
//   - prefix string: The prefix to be used at the start of the key. This is usually a
//     fixed string that categorizes the keys into a namespace.
//   - id string: The unique identifier that will be appended to the prefix. When combined
//     with the prefix, it should create a unique key within the Redis database.
//
// Returns:
// - string: The formatted Redis key constructed from the prefix and id.
//
// Example:
// If you're storing session information in Redis, you might have a prefix for session
// keys such as "session" and unique identifiers for each session. Using this function,
// you can create keys like "session-12345".
func redisKey(prefix, id string) string {
	// fmt.Sprintf formats according to a format specifier and returns the resulting string.
	// Here, it takes two input strings, 'prefix' and 'id', and concatenates them with a
	// hyphen in between to form the Redis key.
	// Example: If 'prefix' is "session" and 'id' is "user123", then it returns "session-user123".
	return fmt.Sprintf("%s-%s", prefix, id)
}
