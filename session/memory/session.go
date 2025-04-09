package memory

import (
	"context"
	"sync"
	"time"

	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/session"
	"github.com/patrickmn/go-cache"
)

// Store represents a storage mechanism for session information. It provides thread-safe
// access to session data and ensures that session information is stored and retrieved
// efficiently with an automated expiration policy.
//
// Fields:
//   - mutex sync.RWMutex: A read/write mutual exclusion lock. It is used to ensure that
//     only one goroutine can write to the sessions cache at a time and that there can be
//     multiple concurrent readers. This ensures that the sessions cache is safe to use
//     concurrently across multiple goroutines without data race conditions.
//   - sessions *cache.Cache: A pointer to an instance of a cache that stores session data.
//     This cache library typically provides a fast in-memory key:value store with
//     expiration capabilities for stored items. The pointer allows the Store to manage
//     sessions in a centralized manner.
//   - expiration time.Duration: A duration after which a session is considered expired
//     and can be removed from the cache. This value sets a time limit on how long
//     session data should persist in the cache before being automatically deleted.
type Store struct {
	// mutex prevents race conditions when accessing the sessions cache by enforcing
	// exclusive write locks and allowing concurrent read locks.
	mutex sync.RWMutex

	// sessions is an in-memory cache that holds session information. Each entry in
	// this cache represents a unique session with its associated data.
	sessions *cache.Cache

	// expiration specifies the duration for which the session data is valid. After this
	// duration, the session data is considered stale and may be purged from the cache.
	expiration time.Duration
}

// InitStore initializes and returns a new Store instance with the specified expiration duration.
//
// The Store instance created by this function is prepared to manage session data with an
// automated expiration policy, which purges the session data after the specified duration.
// A store instance provides a thread-safe way to interact with session data across multiple
// goroutines.
//
// Parameters:
//   - expiration time.Duration: The duration after which sessions should expire and be removed
//     from the cache. This duration dictates how long a session will be kept in memory before
//     being deleted automatically.
//
// Returns:
//   - *Store: A pointer to a newly created Store instance, which holds the session cache and
//     the expiration policy for session data.
//
// Example:
// To create a store with a 30-minute expiration period for sessions, call InitStore with
// time.Minute * 30 as the parameter.
func InitStore(expiration time.Duration) *Store {
	// Create a new Store instance with a cache that has a specific expiration time.
	// Each session will last for the duration of 'expiration' before being automatically
	// removed from the cache.
	return &Store{
		// The cache.New function is invoked with the expiration duration and a cleanup
		// interval of one second. The cleanup interval specifies how often the cache
		// should check for expired items to remove.
		sessions: cache.New(expiration, time.Second),

		// the expiration field is set to the duration passed into the function, ensuring all
		// sessions adhere to this expiration policy.
		expiration: expiration,
	}
}

// NewStore creates a new memory store for session data. It initializes the store
// with a default expiration duration.
func NewStore() (*Store, error) {
	// default expiration of 30 minutes
	return &Store{
		sessions:   cache.New(30*time.Minute, time.Minute),
		expiration: 30 * time.Minute,
	}, nil
}

// Generate creates a new session with the specified ID and stores it in the Store's session
// cache. It ensures that the session is safely created and stored even when accessed by
// multiple goroutines simultaneously.
//
// The function locks the Store's mutex before creating the new session to prevent
// concurrent write access, providing thread safety. It then adds the session to the
// store's cache with the predefined expiration policy before returning the session to the
// caller.
//
// Parameters:
//   - ctx context.Context: The context in which the session is generated. The context
//     allows for controlling cancellations and timeouts, but is not utilized in this
//     function.
//   - id string: The unique identifier for the new session.
//
// Returns:
//   - session.Session: The newly created session object with the provided ID.
//   - error: Any errors encountered during the generation of the session; returns nil
//     since this implementation does not produce errors.
//
// This method may need to be updated if error-handling logic is introduced, such as when
// checks for existing session IDs need to be implemented or if the cache.Set method could
// potentially return an error.
func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	// Lock the mutex to ensure exclusive access to the sessions cache while a new session
	// is being generated and added to the cache. This prevents data races on writes.
	s.mutex.Lock()
	// Unlock the mutex when the function finishes. Defer is used to ensure that unlock
	// happens at the end of the function execution, even if an error occurs or a panic is
	// triggered, which provides safety against deadlocks.
	defer s.mutex.Unlock()

	// Initialize a new Session with the given ID and an empty concurrent-safe map
	// to store session values.
	sess := &Session{
		id:         id,           // Set the session ID to the provided 'id'.
		values:     sync.Map{},   // Initialize a concurrent-safe map for storing session values.
		expiration: s.expiration, // Set the session expiration time.
		modified:   false,        // Initialize the modified flag as false.
	}

	// Add the newly created session to the cache with the Store's expiration duration
	// policy, so it gets automatically evicted from cache when it expires.
	s.sessions.Set(id, sess, s.expiration)

	// Return the new session and nil since no error can occur in the current implementation.
	return sess, nil
}

// Refresh updates the expiration time of an existing session, effectively "refreshing" it.
// This method looks up a session by its ID and, if found, extends its life in the Store's
// cache according to the Store's expiration policy.
//
// The method provides thread safety by locking the Store's mutex, thereby preventing
// concurrent access to the sessions cache during the update process.
//
// Parameters:
//   - ctx context.Context: The context in which the session refresh is executed. The context
//     provides the ability to handle cancellations and timeouts, although this functionality
//     is not utilized in the current function implementation.
//   - id string: The unique identifier of the session that is being refreshed.
//
// Returns:
//   - error: An error is returned if the session with the specified ID cannot be found;
//     otherwise, nil is returned after successfully refreshing the session expiration time.
//
// This method assumes that the errs.ErrIdSessionNotFound function returns a relevant
// error when a session with a given ID does not exist.
func (s *Store) Refresh(ctx context.Context, id string) error {
	// Lock the mutex to ensure exclusive access to the sessions cache during the refresh
	// operation. This is crucial to prevent any concurrent write or read operations that
	// might affect the consistency of the data.
	s.mutex.Lock()
	// Defer the Unlock operation to ensure that the mutex is always released at the end of
	// the method's execution, regardless of where the function exits, which prevents
	// potential deadlocks.
	defer s.mutex.Unlock()

	// Attempt to retrieve the session with the specified ID from the cache.
	val, ok := s.sessions.Get(id)
	if !ok {
		// If the session is not found, return an error indicating the session ID does not
		// exist, using a custom error function assumed to be defined in the 'errs' package.
		return errs.ErrIdSessionNotFound()
	}

	// If the session is found, reset its expiration time in the cache using the
	// predefined expiration duration of the Store.
	s.sessions.Set(id, val, s.expiration)

	// Return nil as no errors occurred during the refresh operation.
	return nil
}

// Remove deletes a session from the Store's session cache using the provided session ID.
// It's designed to ensure thread-safe deletion of session data, avoiding concurrent access issues.
//
// Parameters:
//   - ctx context.Context: The context in which session removal is requested. This
//     typically contains information about deadlines, cancellation signals, and other
//     request-scoped values relevant to the operation. However, the context is not directly
//     utilized within this function.
//   - id string: The unique identifier of the session to be removed from the cache.
//
// Returns:
//   - error: An error is returned if any issues occur during the deletion process;
//     however, in this implementation, the function always returns nil, indicating success.
//
// It's important to handle potential errors in future revisions of this method, particularly
// when operations that can fail are introduced.
func (s *Store) Remove(ctx context.Context, id string) error {
	// Lock the mutex to prevent other goroutines from modifying the session cache
	// concurrently, guaranteeing the safe deletion of a session.
	s.mutex.Lock()
	// Unlock the mutex when the function is complete. Using defer ensures the mutex is
	// always released, even if an error or panic occurs, preventing potential deadlocks.
	defer s.mutex.Unlock()

	// Check if the session exists in the cache before attempting to delete it.
	// This step is not strictly necessary since `Delete` is a no-op if the key doesn't
	// exist, but it may be useful for auditing or debugging purposes.
	_, exists := s.sessions.Get(id)
	if !exists {
		// If the session does not exist, there's nothing to delete. In this implementation,
		// we choose to return nil to indicate that the operation was successful (the
		// session doesn't exist as requested).
		return nil
	}

	// Delete the session from the cache. Since the cache is a map-like structure, the
	// deletion operation is a simple removal of the key-value pair with the given ID.
	s.sessions.Delete(id)

	// Return nil to indicate successful removal of the session.
	return nil
}

// Get retrieves a session from the Store's session cache using the provided session ID.
// It provides thread-safe access to session data through read-locking mechanisms.
//
// Parameters:
//   - ctx context.Context: The context in which the session retrieval is requested. The context
//     can carry deadlines, cancellation signals, and other request-scoped values across API
//     boundaries. However, the method does not utilize these context features in its current
//     implementation.
//   - id string: The unique identifier used to look up the session in the session cache.
//
// Returns:
//   - session.Session: The retrieved session object if found.
//   - error: An error is returned when the session with the specified ID cannot be found or
//     if there are issues during the retrieval process.
//
// The method assumes that the errs.ErrIdSessionNotFound function returns an appropriate
// error when a session ID doesn't exist in the cache.
func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	// Acquire a read lock to allow concurrent reads but prevent concurrent writes, ensuring
	// that the session cache remains consistent during the read operation.
	s.mutex.RLock()
	// Release the read lock when the function returns. Using defer ensures the lock is
	// always released, even if an error occurs or the function returns early.
	defer s.mutex.RUnlock()

	// Attempt to retrieve the session with the given ID from the session cache.
	val, ok := s.sessions.Get(id)
	if !ok {
		// If the session is not found in the cache, return an error indicating that the
		// session with the given ID does not exist.
		return nil, errs.ErrIdSessionNotFound()
	}

	// If the session is found, cast it to the session.Session type and return it along
	// with a nil error to indicate successful retrieval.
	return val.(session.Session), nil
}

// GC performs garbage collection on expired sessions in the session store.
// This method triggers the cache's internal garbage collection mechanism to
// clean up expired sessions, freeing up memory and resources.
//
// Parameters:
//   - ctx context.Context: The context in which the garbage collection is requested.
//     This context can be used to control the execution of the garbage collection,
//     but is not utilized in the current implementation.
//
// Returns:
//   - error: An error if the garbage collection operation fails; otherwise, nil
//     indicating success.
//
// Note that go-cache has its own internal garbage collection that runs on a separate
// goroutine, so this method primarily forces an immediate clean-up cycle. For most
// applications, relying on the automatic garbage collection is sufficient.
func (s *Store) GC(ctx context.Context) error {
	// Lock the store to prevent concurrent operations during garbage collection
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Force the cache to delete all expired items immediately
	s.sessions.DeleteExpired()

	return nil
}

// Session represents an in-memory session object that stores and retrieves
// user-specific data with a unique identifier. It implements the session.Session
// interface and provides thread-safe operations on session data.
//
// Fields:
//   - id: A unique identifier for the session, used to reference it across HTTP requests.
//   - values: A thread-safe map that stores key-value pairs of session data.
//   - modified: A flag to track whether the session has been modified since creation.
//   - expiration: The duration for which the session is valid.
//   - maxAge: Maximum lifetime of the session in seconds.
type Session struct {
	// id holds the unique identifier for the session, which is typically generated when
	// a new session is created and persists until the session is ended or times out.
	id string

	// values is a thread-safe map provided by the sync package. It is optimized for
	// scenarios where the entry is only written once but read many times, as it's
	// common with session data.
	values sync.Map

	// modified indicates whether the session has been modified
	modified bool

	// expiration specifies how long the session is valid for
	expiration time.Duration

	// maxAge specifies the maximum lifetime of the session in seconds
	maxAge int
}

// Get retrieves a value from the session based on the provided key.
// This method provides thread-safe access to session data.
//
// Parameters:
//   - ctx context.Context: The context in which the retrieval is requested. The context
//     can carry deadlines, cancellation signals, and other request-scoped values across API
//     boundaries. However, this method does not utilize these context features.
//   - key string: The identifier used to retrieve the corresponding value from the session's
//     data store.
//
// Returns:
//   - any: The retrieved value associated with the key. If the key is not found, nil is returned.
//   - error: An error if the retrieval operation fails; otherwise, nil is returned.
//
// This method is safe for concurrent use by multiple goroutines as it uses the thread-safe
// sync.Map to store session values.
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	// Attempt to load the value associated with the given key from the session's data store.
	// The Load method of sync.Map provides thread-safe access to the map's contents.
	value, ok := s.values.Load(key)
	if !ok {
		// If the key is not found, return nil without an error. This allows callers to
		// distinguish between a key not present (nil, nil) and an error condition (nil, err).
		return nil, nil
	}

	// Return the value associated with the key and nil for the error to indicate successful retrieval.
	return value, nil
}

// Set stores a value in the session using the provided key. If the key already exists,
// the value is updated. This method ensures thread-safe modification of session data.
//
// Parameters:
//   - ctx context.Context: The context in which the storage is requested. The context
//     can carry deadlines, cancellation signals, and other request-scoped values across API
//     boundaries. However, this method does not utilize these context features.
//   - key string: The identifier under which the value should be stored in the session's
//     data store.
//   - value any: The data to be stored in the session. This can be of any type.
//
// Returns:
//   - error: An error if the storage operation fails; otherwise, nil is returned.
//
// This method is safe for concurrent use by multiple goroutines as it uses the thread-safe
// sync.Map to store session values.
func (s *Session) Set(ctx context.Context, key string, value any) error {
	// Store the key-value pair in the session's data store. The Store method of sync.Map
	// provides thread-safe access for writing to the map.
	s.values.Store(key, value)

	// Mark the session as modified
	s.modified = true

	return nil
}

// Delete removes a key-value pair from the session. If the key does not exist,
// this operation is a no-op.
//
// Parameters:
//   - ctx context.Context: The context in which the deletion is requested. This context
//     is not used in the current implementation but is included for interface compliance.
//   - key string: The identifier of the key-value pair to be removed from the session.
//
// Returns:
//   - error: An error if the deletion operation fails; otherwise, nil is returned.
//
// This method is safe for concurrent use by multiple goroutines.
func (s *Session) Delete(ctx context.Context, key string) error {
	// Delete the key from the session's data store
	s.values.Delete(key)

	// Mark the session as modified
	s.modified = true

	return nil
}

// ID returns the unique identifier of the session. This identifier is used to
// associate the session with a particular user or client across multiple HTTP requests.
//
// Returns:
//   - string: The unique identifier of the session.
//
// This method is safe for concurrent use by multiple goroutines.
func (s *Session) ID() string {
	// Simply return the session's ID field.
	return s.id
}

// Save persists any changes made to the session. In this in-memory implementation,
// there is no need for explicit persistence, so this method simply resets the
// modified flag.
//
// Returns:
//   - error: An error if the save operation fails; otherwise, nil is returned to
//     indicate success.
//
// This method is safe for concurrent use by multiple goroutines.
func (s *Session) Save() error {
	// Reset the modified flag to indicate that the session has been saved
	s.modified = false
	return nil
}

// IsModified returns whether the session has been modified since it was
// last saved or created.
//
// Returns:
//   - bool: true if the session has been modified, false otherwise.
//
// This method is safe for concurrent use by multiple goroutines.
func (s *Session) IsModified() bool {
	return s.modified
}

// SetMaxAge sets the maximum lifetime of the session in seconds.
//
// Parameters:
//   - seconds: The maximum lifetime of the session in seconds. A positive value
//     sets the session to expire after the specified number of seconds. A negative
//     value means the session expires when the browser is closed. A zero value
//     deletes the session immediately.
//
// This method is safe for concurrent use by multiple goroutines.
func (s *Session) SetMaxAge(seconds int) {
	s.maxAge = seconds

	// If maxAge is set to 0 or negative, we might want to adjust the expiration
	if seconds <= 0 {
		// For in-memory sessions, we could set a very short expiration time
		// for sessions that should be deleted immediately
		if seconds == 0 {
			s.expiration = time.Second
		}
		// For sessions that should expire when the browser closes,
		// we keep the default expiration
	} else {
		// For positive values, set the expiration to the specified number of seconds
		s.expiration = time.Duration(seconds) * time.Second
	}

	// Mark the session as modified
	s.modified = true
}
