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
		id:     id,         // Set the session ID to the provided 'id'.
		values: sync.Map{}, // Initialize a concurrent-safe map for storing session values.
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
	// released even if an unexpected panic occurs, preventing deadlocks.
	defer s.mutex.Unlock()

	// Delete the session with the given ID from the cache. The Delete method does not return
	// an error, so there's no error handling needed here. If error possibilities are introduced
	// in future implementations, they should be handled accordingly.
	s.sessions.Delete(id)

	// Return nil to indicate that the session has been successfully removed.
	return nil
}

// Get retrieves the session associated with the provided ID from the store.
// It features thread safety by using a read lock to allow multiple concurrent
// read operations while preventing write operations, ensuring data consistency.
//
// Parameters:
//   - ctx context.Context: The context in which the session retrieval is taking
//     place. The context could be used to carry deadlines, cancellation signals,
//     and other request-scoped values, but it's not used within this function.
//   - id string: The unique identifier for the session we want to retrieve.
//
// Returns:
// - session.Session: The session object associated with the ID, if found.
// - error: An error if no session is found with the given ID, otherwise nil.
//
// If future versions of the Get function introduce returnable errors in other
// situations, this documentation and implementation may need to be updated
// accordingly.
func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	// Use a read lock (RLock) to allow for concurrent read access to the sessions
	// cache by multiple goroutines, while still preventing any writes, maintaining
	// data consistency and integrity.
	s.mutex.RLock()
	// Ensure the read lock is released after this function's execution completes.
	// This is essential to ensure that it doesn't block subsequent write operations.
	defer s.mutex.RUnlock()

	// Attempt to retrieve the session using the provided ID.
	sess, ok := s.sessions.Get(id)
	if !ok {
		// If the session is not found, return an appropriate error. Assumes that
		// errs.ErrSessionNotFound is a function returning a standardized error message.
		return nil, errs.ErrSessionNotFound()
	}

	// Assert that the interface{} type returned from sessions.Get is indeed a *Session.
	// This is necessary because the underlying data structure is generic and can store
	// various types.
	return sess.(*Session), nil
}

// Session is a data structure that represents a user session in a concurrent environment.
// It stores session-specific information, such as a unique session ID and session values,
// in a thread-safe manner, ensuring that multiple goroutines can interact with the values
// concurrently without causing race conditions.
//
// Fields:
//   - id string: The unique identifier of the session, used to retrieve or differentiate the session.
//   - values sync.Map: A thread-safe map provided by the sync package, used to store session-specific
//     values. sync.Map uses fine-grained locking which is more efficient for cases where
//     multiple goroutines are reading, writing, and iterating over entries in the map simultaneously.
type Session struct {
	// id holds the unique identifier for the session, which is typically generated when
	// a new session is created and persists until the session is ended or times out.
	id string

	// values is a thread-safe map provided by the sync package. It is optimized for
	// scenarios where the entry is only written once but read many times, as it's
	// common with session data.
	values sync.Map
}

// Get retrieves a value from the session based on a provided key. It utilizes the sync.Map's Load method
// to safely access the value, ensuring that simultaneous reads/writes to the values Map are properly synchronized.
//
// Parameters:
//   - ctx context.Context: The context for the operation. While it is a common pattern to include context
//     for potential future use in cancellations and timeouts, it is not currently used in this method.
//   - key string: The key associated with the value to retrieve from the session.
//
// Returns:
//   - any: The value associated with the key if it is found within the session's values map.
//   - error: An error if the key does not exist in the session's values map. ErKeyNotFound is returned
//     with the missing key as its parameter.
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	// Attempt to retrieve the value from the session's values map using the provided key.
	val, ok := s.values.Load(key)

	// Check if the key was found in the map.
	if !ok {
		// If the key is not found, use errs.ErrKeyNotFound to return an error with the key.
		// ErrKeyNotFound should be a predefined error type in the errs package that encapsulates
		// the error scenario where a key is not found within the map.
		return nil, errs.ErrKeyNotFound(key)
	}

	// If the key is found, return the corresponding value.
	return val, nil
}

// Set assigns a value to a specific key within the session. It leverages the Store method
// from sync.Map for a safe concurrent write operation. As of this implementation, it always
// returns a nil error, denoting a successful operation.
//
// Parameters:
//   - ctx context.Context: The context for the operation. Although it is often included to handle
//     request-scoped values, cancellations, and timeouts, it is not utilized in this method at present.
//   - key string: The key to associate with the value within the session's values map.
//   - value any: The value to be associated with the key.
//
// Returns:
//   - error: nil, indicating that the value was successfully stored. If future implementations introduce
//     potential errors, the method's signature and documentation would need corresponding updates.
func (s *Session) Set(ctx context.Context, key string, value any) error {
	// Store the value in the session's values map with the specified key. The Store method ensures
	// that the write operation is safe to use concurrently with other reads and writes.
	s.values.Store(key, value)

	// As per the current implementation, there are no failure scenarios that would result in
	// an error being returned. Thus, nil is returned to indicate success.
	return nil
}

// ID returns the unique session identifier for this session.
func (s *Session) ID() string {
	return s.id
}

// Save saves any changes to the session.
// For memory sessions, this is a no-op as changes are saved immediately.
func (s *Session) Save() error {
	// Memory session values are saved immediately when Set is called
	// so no additional saving is needed
	return nil
}
