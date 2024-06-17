package session

import (
	"github.com/dormoron/mist"
	"github.com/google/uuid"
)

// The Manager struct acts as a centralized component that orchestrates the session management
// within a web application. By embedding the Store and Propagator interfaces, it seamlessly combines
// the functionality of session storage and identifier propagation. The Manager handles the complete lifecycle
// of a session, which includes creating sessions, storing them, transmitting session identifiers in HTTP
// messages, and cleaning up as needed. This higher-level struct allows for simplified session management
// throughout the application, as it encapsulates all the necessary operations within a single entity.
//
// Fields:
//   - CtxSessionKey: A string that represents the key under which the session object is stored in
//     the context of an HTTP request. This allows middleware and handlers to retrieve the session
//     info from the context using this key, facilitating a standard way of accessing session data
//     during the processing of a request.
//
// The inclusion of both the Store and Propagator interfaces suggests that any instance of Manager is
// capable of performing all session-related operations defined by these interfaces. This includes generating
// and managing session data (through the Store interface) and handling the session identifiers across
// HTTP requests and responses (through the Propagator interface).
//
// Here is an example of how the Manager struct could be initialized and used within an application:
//
//	func main() {
//	    // Initialize the session manager with specific implementations of Store and Propagator.
//	    sessionManager := &Manager{
//	        Store: NewRedisStore(), // assuming NewRedisStore returns an implementation of Store
//	        Propagator: NewCookiePropagator("session_id"), // assuming NewCookiePropagator returns an implementation of Propagator
//	        CtxSessionKey: "session", // the key used to store session objects in context
//	    }
//
//	    // Set up your HTTP server, routes, middleware, etc.,
//	    // and use sessionManager to manage sessions in your application.
//	}
//
// Through Manager, all handlers and middleware in the application can interact with sessions
// using a standardized interface without worrying about the underlying storage or communication
// mechanisms, which are abstracted away by the implementations of Store and Propagator.
//
// Implementers of the Manager struct should ensure that necessary synchronizations or concurrent
// access handling are considered in their implementations of Store and Propagator to prevent race
// conditions or data inconsistencies.
type Manager struct {
	Store                // Handles storage and retrieval of session data.
	Propagator           // Manages transmission of session identifiers in HTTP messages.
	CtxSessionKey string // Key for session object storage in request context.
}

// GetSession is a method that retrieves the current user's session from the HTTP request
// and caches it in the context for future use within the scope of the current request processing.
// This method provides a single entry point for session retrieval, and ensures that the session
// is loaded only once per request, thereby improving performance and reducing redundant operations.
//
// The flow is as follows:
//  1. It first checks if the UserValues map within the mist.Context is initialized. If not,
//     it initializes the map to store the session object later in the process.
//  2. The method then tries to retrieve the session from the UserValues map using the CtxSessionKey
//     defined in the Manager struct. This is to check if the session was already fetched and cached
//     earlier in the current request lifecycle.
//  3. If the session is found in the map, it is returned immediately, avoiding any further operations.
//  4. If not, the method utilizes the Propagator interface's Extract method to retrieve the session
//     identifier from the incoming HTTP request, which is typically read from a cookie or request header.
//  5. With the session identifier obtained, the method then fetches the actual session data using the
//     Store interface's Get method. This method call also passes along the context from the request to handle
//     any session-related context operations such as deadlines or cancellations.
//  6. After the session is successfully retrieved, it is stored in the UserValues map using the CtxSessionKey
//     for quick access during subsequent calls within the same request lifecycle.
//  7. Finally, the actual session data or an error (if any occurred while retrieving the session identifier
//     or the session data) is returned.
//
// If at any point there is a failure to retrieve the session identifier or the session data, an error
// is returned to the caller. This method centralizes error handling related to session retrieval, which
// simplifies the session logic elsewhere in the application.
//
// The mist.Context is assumed to be a custom HTTP context that contains both the standard library context
// and additional data fields used for managing user-specific values within a single request lifecycle.
//
// Usage:
// This method should be called by middlewares or handlers that require access to the current user's session.
// It exempts them from having to handle low-level session extraction and storage mechanisms directly.
func (m *Manager) GetSession(ctx *mist.Context) (Session, error) {
	// Ensure the map used to store values in the context is initialized.
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}

	// Attempt to retrieve the session from the cache in the user values map.
	val, ok := ctx.UserValues[m.CtxSessionKey]
	if ok {
		return val.(Session), nil
	}

	// Session not found in cache, so extract the session ID from the HTTP request.
	sessId, err := m.Propagator.Extract(ctx.Request)
	if err != nil {
		return nil, err
	}

	// Retrieve the session data using the extracted session ID.
	session, err := m.Store.Get(ctx.Request.Context(), sessId)
	if err != nil {
		return nil, err
	}

	// Store the session in the map for quick access during this request lifecycle.
	ctx.UserValues[m.CtxSessionKey] = session
	return session, nil
}

// InitSession is responsible for creating a new session and associating it with the client who initiated
// the HTTP request. It is typically called when a new user visits the application and a new session needs to
// be established. The method leverages the capabilities of the embedded interfaces within the Manager struct
// to generate a unique session identifier, create a new session, and transmit this session identifier back to
// the client for future interactions.
//
// The process involves the following steps:
//  1. Generate a new unique identifier for the session using a universally unique identifier (UUID) library.
//  2. With the new session identifier, the method calls the Generate method of the embedded Store interface
//     to actually create a new session in the session store. This session creation is supposed to associate
//     the generated UUID with a new session object and store it in whatever storage mechanism the Store
//     interface implementation uses (e.g., in-memory, database, etc.). The request context is provided
//     to handle any necessary context operations such as deadlines or request cancellations.
//  3. If an error occurs during session generation (e.g., database error, context deadline exceeded), this
//     error is returned to the caller and no further steps are taken.
//  4. Should the session generation be successful, the new identifier is then propagated to the client using
//     the Inject method of the Propagator interface which is part of the Manager. This step typically involves
//     setting a cookie or an HTTP header in the response so that the client can include this identifier in
//     subsequent requests to maintain the session context.
//  5. The new session object is returned to the caller along with any error that might occur during the
//     identifier injection process (though no error is expected in creating a new session at this point,
//     errors might occur while setting an HTTP response header or cookie).
//
// It's important for the implementer to note that after this method is called, the client must include
// the session identifier in subsequent requests, and the server will need to handle this identifier to
// retrieve the associated session from the store.
//
// The mist.Context parameter provides request-specific information including the Request and
// ResponseWriter which are used to retrieve and set information related to the session. This context
// is assumed to be part of a custom processing pipeline that allows easy access and manipulation of
// HTTP request and response data.
//
// Usage:
// This method should be called when a new user session needs to be initiated. Typically, it would be
// invoked within the auth process, or when a session is not found for a request and needs
// to be created.
//
// Example:
//
//	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
//	    ctx := mist.NewContext(r, w)
//	    session, err := sessionManager.InitSession(ctx)
//	    if err != nil {
//	        // Handle error
//	    }
//	    // Session initialized successfully, store session data or modify response as needed.
//	})
func (m *Manager) InitSession(ctx *mist.Context) (Session, error) {
	// Generate a new UUID for the session.
	id := uuid.New().String()

	// Create a new session with the generated UUID.
	sess, err := m.Generate(ctx.Request.Context(), id)
	if err != nil {
		return nil, err // Return error if session generation fails.
	}

	// Propagate the new session identifier to the client using the ResponseWriter.
	err = m.Inject(id, ctx.ResponseWriter)
	return sess, err // Return the new session and any error from identifier propagation.
}

// RefreshSession is a method that updates an existing session's expiry time to extend its life.
// This is commonly referred to as "session refresh" or "session regeneration" and is a critical
// security practice to prevent "session fixation" attacks. It is typically used in scenarios where
// the application wants to ensure that the session remains valid, such as after a user performs
// a sensitive action or after a fixed interval of time.
//
// The session is refreshed using the following process:
//  1. Retrieve the current session associated with the request by calling the GetSession method.
//     This uses the context and the mechanisms provided by the Propagator and Store interfaces
//     to locate the session data.
//  2. If an error is encountered during session retrieval, such as when the session does not exist
//     or the session identifier is invalid, the error is immediately returned and the refresh
//     operation is aborted.
//  3. Assuming the session is retrieved successfully, the method proceeds to refresh the session's
//     expiry time by calling the Refresh method of the Store interface. The Store interface's Refresh
//     method is implemented by the session storage mechanism (e.g., a database) and is responsible for
//     updating the session's expiry time within the storage backend.
//  4. The Refresh method takes the session ID obtained from the retrieved session and the request context
//     (to allow for timeout or cancelation) as parameters.
//  5. An error is returned if the attempt to refresh the session fails; otherwise, nil is returned,
//     indicating a successful refresh.
//
// It is important to design the underlying Store implementation to handle the refresh operation atomically
// to avoid any race conditions or inconsistencies.
//
// The mist.Context is assumed to be a custom object that encapsulates the standard Go context and additional
// information pertaining to the HTTP request cycle (for example, HTTP request and response accessor methods).
//
// Usage:
// This method should be used when you need to prolong a user's session after some event or at regular intervals
// during their interaction with the application. It is most commonly placed within middleware or wrapped within
// handler functions that trigger the refresh logic.
//
// Here's an example of how to use RefreshSession within an HTTP handler function:
//
//	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
//	    ctx := mist.NewContext(r, w)
//	    err := sessionManager.RefreshSession(ctx)
//	    if err != nil {
//	        // Handle error (e.g., redirect to login page)
//	    }
//	    // Proceed with handling the request, knowing the session has been refreshed.
//	})
func (m *Manager) RefreshSession(ctx *mist.Context) error {
	// Retrieve the existing session.
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err // Return error if session retrieval fails.
	}

	// Refresh the session's expiry time in the store.
	return m.Refresh(ctx.Request.Context(), sess.ID())
	// Any error during refresh is returned to the caller.
}

// RemoveSession is a method designed to delete a user's session from the session store
// and to clear any session identifiers from the client's context, effectively logging
// the user out. This can be a critical function for user security, ensuring sessions are
// properly terminated when a user logs out or when their session should be invalidated for
// other reasons, such as after changing a password, after a period of inactivity, or for
// administrative logout purposes.
//
// The flow of the session removal process operates as follows:
//  1. Attempt to retrieve the existing session from the mist.Context by calling the GetSession method
//     of the Manager struct, which retrieves session data based on a session identifier found in the
//     client's request.
//  2. If this retrieval fails (for example, if the session has already expired or does not exist), an
//     error is returned immediately and no further action is taken.
//  3. Once the session is successfully retrieved, the Manager struct's embedded Store interface is used
//     to remove the session data from the persistent session storage via the Store.Remove method. This
//     requires the context from the current HTTP request (for deadline or cancellation purposes) and the
//     session ID.
//  4. An error from the Store.Remove operation (which might indicate that the session data could not be
//     deleted, for instance, due to a database error) is also returned immediately, preventing the
//     process from continuing.
//  5. Assuming the session has been removed from storage without errors, the Manager struct's embedded
//     Propagator interface performs the final step of the process. The Propagator.Remove method is called
//     to clear any session identifiers from the client's environment, such as by clearing the client's
//     session cookie or other client-side storage. This ensures that the session cannot be reused.
//  6. Finally, the method returns nil to indicate that the session was successfully removed, or it returns
//     an error if the Propagator.Remove operation failed.
//
// The input parameter mist.Context is a custom HTTP context type which provides access to the HTTP request
// and response objects along with additional utilities. This method manipulates both the response to clear
// client identifiers and the session store for backend operations.
//
// Usage:
// This method is called when a session needs to be unequivocally terminated, such as during a logout process.
// It is often paired with auth middleware or within an HTTP handler that processes logout requests.
//
// Example usage within an HTTP handler:
//
//	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
//	    ctx := mist.NewContext(r, w)
//	    if err := sessionManager.RemoveSession(ctx); err != nil {
//	        // Handle errors: maybe log the error and redirect to the home page?
//	    }
//	    // Successful removal: redirect to login page or confirm logout.
//	})
func (m *Manager) RemoveSession(ctx *mist.Context) error {
	// First, retrieve the current session from the context.
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err // If the session cannot be retrieved, return the error immediately.
	}

	// Remove the session from the store using the session ID.
	err = m.Store.Remove(ctx.Request.Context(), sess.ID())
	if err != nil {
		return err // If there's an error removing the session from the store, return the error.
	}

	// Remove the session identifier from the client's context.
	return m.Propagator.Remove(ctx.ResponseWriter)
	// Return any errors from removing the session identifier or nil if successful.
}
