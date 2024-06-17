package session

import (
	"context"
	"net/http"
)

// The Store interface defines a set of methods for session management in an application.
// This interface requires any session store implementation to create, refresh, remove,
// and retrieve sessions. A session typically represents a period of interaction between
// a user and an application and can hold crucial data such as user identity, preferences, and
// application state. The operations to manipulate these sessions are context-aware, meaning they
// can be cancelled, have timeouts attached to them, or carry request-scoped values via the provided
// context.Context object. This interface allows an application to abstract away the details of how
// sessions are stored and managed, whether it's in memory, a database, a file system, or some other
// form of storage.
// Methods:
//   - Generate(ctx context.Context, id string) (Session, error): Responsible for creating a new session
//     with the given identifier 'id'. It returns the newly created Session object and any error encountered
//     during the creation process. This is typically called during user login or initial interaction.
//   - Refresh(ctx context.Context, id string) error: Updates an existing session's expiration or last active time,
//     if applicable, to prevent the session from expiring. This is typically used for extending user sessions to
//     keep them logged in or maintaining their state.
//   - Remove(ctx context.Context, id string) error: Deletes a session from the store, effectively logging out
//     the user or clearing any state stored in the session. This occurs when a user explicitly logs out or when
//     a session needs to be invalidated for security reasons.
//   - Get(ctx context.Context, id string) (Session, error): Retrieves the session associated with the given
//     identifier 'id'. It returns the Session object if it exists and any error that occurs during the retrieval.
//     This is called whenever an application needs to access the session data for a request.
//
// The 'Session' type mentioned in the methods is expected to be an interface or a struct that encapsulates the
// session data. The specific implementation of 'Session' will depend on the application's requirements.
// Note that this interface does not specify how to handle session collision or the specifics of session expiration
// details. Implementations of this interface should ensure these aspects are handled according to their needs and
// document any specific behaviors.
// Example Implementation:
// An in-memory store could implement the Store interface like so:
//
//	type InMemoryStore struct {
//	    sessions map[string]Session
//	    lock sync.RWMutex
//	    // ... additional fields and methods ...
//	}
//
//	func (s *InMemoryStore) Generate(ctx context.Context, id string) (Session, error) {
//	    // ... generate and store a new session ...
//	}
//
//	func (s *InMemoryStore) Refresh(ctx context.Context, id string) error {
//	    // ... refresh session expiration time ...
//	}
//
//	func (s *InMemoryStore) Remove(ctx context.Context, id string) error {
//	    // ... remove session from store ...
//	}
//
//	func (s *InMemoryStore) Get(ctx context.Context, id string) (Session, error) {
//	    // ... retrieve a session based on the id ...
//	}
//
// The above interface abstraction enables one to switch session store implementations with minimal changes to
// the overall application logic, providing flexibility and scalability for session management strategies.
type Store interface {
	Generate(ctx context.Context, id string) (Session, error) // Create a new session
	Refresh(ctx context.Context, id string) error             // Extend a session's life
	Remove(ctx context.Context, id string) error              // Delete an existing session
	Get(ctx context.Context, id string) (Session, error)      // Retrieve a session's data
}

// Session is an interface that defines the contract for a session management system.
// In web applications, a session represents a single user's interactions with the application
// across multiple requests. It is used to store and retrieve data specific to a user or session scope.
// The Session interface allows for getting and setting of session values and retrieval of a unique
// session identifier. Session implementations should handle concurrency and provide some form of
// persistence mechanism, which could range from in-memory storage to database-backed solutions.
// These operations are context-aware, allowing them to participate in context-specific lifecycles,
// such as request timeouts or cancellations.
// Methods:
//   - Get(ctx context.Context, key string) (any, error): Retrieves a value from the session data.
//     The 'key' argument specifies which value to retrieve. If the key does not exist, the method
//     should return nil without error. An error is returned only if there is a problem with the
//     retrieval process itself, not if the key is merely absent.
//   - Set(ctx context.Context, key string, value any) error: Assigns a 'value' to a 'key' in the session data.
//     If the 'key' already exists, the value should be overwritten. As with 'Get', any context-related
//     behavior should be handled within this method. If there is an error while setting the value (e.g.,
//     write failure), an error should be reported back to the caller.
//   - ID() string: This method returns the unique identifier for the session. Typically, this identifier
//     is generated when the session is first created and remains constant throughout the session's
//     lifecycle. The ID is used to link a session to a specific user or interaction sequence.
//
// Usage and Implementation Notes:
// The 'Session' type assumes 'any' type for stored values, allowing for flexibility in what kinds of
// data can be stored in the session. However, the underlying implementation needs to ensure proper
// serialization and deserialization of these values as sessions may be persisted across different
// requests and even server restarts.
// It is important to ensure thread safety within the methods if the session store is accessed
// concurrently. Sessions should also handle cleanup and invalidation as necessary, and the implementation
// should detail how long session values persist (e.g., expiry time, persistent or ephemeral storage).
// As an example, an implementation could use a synchronized map structure to store session values,
// with an additional expiration timestamp to handle session lifetimes:
//
//	type InMemorySession struct {
//	    id string
//	    values sync.Map  // Thread-safe map to store session values
//	    expiry time.Time  // Expiration time of the session
//	    // ... additional session properties and mutexes ...
//	}
//
//	func (s *InMemorySession) Get(ctx context.Context, key string) (any, error) {
//	    // ... retrieve value from the session ...
//	}
//
//	func (s *InMemorySession) Set(ctx context.Context, key string, value any) error {
//	    // ... set value in the session ...
//	}
//
//	func (s *InMemorySession) ID() string {
//	    return s.id
//	}
//
// The above example provides a simple template for how session data can be managed within a
// web application. It emphasizes the importance of thread-safe data access and context management.
type Session interface {
	Get(ctx context.Context, key string) (any, error)     // Retrieve a value from the session
	Set(ctx context.Context, key string, value any) error // Store a value in the session
	ID() string                                           // Return the session's unique identifier
}

// The Propagator interface defines methods responsible for managing the propagation
// of session identifiers across HTTP requests and responses in a web application.
// This interface abstracts the operations of adding, retrieving, and deleting a session
// identifier (such as a cookie or a auth) to and from HTTP requests and responses, thereby allowing
// session tracking and management universally across different systems and components.
// Methods:
//   - Inject(id string, writer http.ResponseWriter) error: Inserts a session identifier 'id' into
//     an outgoing HTTP response using the provided ResponseWriter. This is typically used to set a
//     cookie or an HTTP header that contains the session identifier, enabling the client (e.g., a web browser)
//     to return the identifier in subsequent requests to maintain the session state. If there's an issue
//     writing to the response (e.g., headers already written), the method should return an error.
//   - Extract(req *http.Request) (string, error): Analyzes an incoming HTTP request and extracts the session
//     identifier. This is commonly used to read a cookie or a header from the request, validating its presence
//     and perhaps its format or integrity. If the session identifier is successfully retrieved, it is returned;
//     otherwise, an error is returned, which may indicate that the session identifier is not present or invalid.
//   - Remove(writer http.ResponseWriter) error: Clears the session identifier from the outgoing HTTP response,
//     effectively terminating the session from the server's perspective. This is generally implemented by
//     invalidating a cookie or clearing a header in the response. If there's a problem modifying the response
//     (e.g., if it's too late to change headers), the method should return an error.
//
// Session identifiers are typically compact and should be handled in a secure manner to prevent security
// vulnerabilities such as Session Hijacking or Session Fixation. Implementers should ensure that identifiers
// are properly encrypted or signed and handled over secure channels when necessary.
// An example implementation might look like:
//
//	type CookiePropagator struct {
//	    // CookieName defines the name of the cookie to be used for session identification.
//	    CookieName string
//	    // Other configuration for the cookie such as Domain, Path, Secure flags, etc.
//	}
//
//	func (cp *CookiePropagator) Inject(id string, writer http.ResponseWriter) error {
//	    // ... set cookie with the session id ...
//	}
//
//	func (cp *CookiePropagator) Extract(req *http.Request) (string, error) {
//	    // ... read and return the session id from the request cookie ...
//	}
//
//	func (cp *CookiePropagator) Remove(writer http.ResponseWriter) error {
//	    // ... invalidate the cookie to remove the session ...
//	}
//
// This interface is essential for ensuring session continuity and coherence in stateful web
// applications, providing a high-level abstraction over the raw HTTP mechanisms used underneath.
type Propagator interface {
	Inject(id string, writer http.ResponseWriter) error // Add a session identifier to the HTTP response
	Extract(req *http.Request) (string, error)          // Retrieve a session identifier from the HTTP request
	Remove(writer http.ResponseWriter) error            // Delete a session identifier from the HTTP response
}
