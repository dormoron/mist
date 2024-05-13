package casbin

import (
	"encoding/json"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/dormoron/mist"
	"github.com/fsnotify/fsnotify"
	"log"
	"net/http"
	"sync"
)

// SubResolver Define a type for a function that resolves a subject (user) from an HTTP request.
// This function must return the resolved subject as a string and any potential error encountered during the process.
type SubResolver func(*http.Request) (string, error)

// MiddlewareBuilder is a struct that assembles middleware components by using the Casbin library for authorization and a file system watcher for policy changes.
type MiddlewareBuilder struct {
	enforcer    *casbin.Enforcer  // An instance of the Casbin Enforcer which will enforce the policies.
	subResolver SubResolver       // The function used to resolve the subject from the HTTP request.
	cacheMutex  sync.RWMutex      // A reader/writer lock to synchronize access to the policy from multiple goroutines.
	policyFile  string            // The path to the policy file that Casbin uses.
	watcher     *fsnotify.Watcher // A file system watcher to detect changes in the policy file.
}

// InitMiddlewareBuilder is a constructor function for MiddlewareBuilder.
// It initializes the Casbin enforcer, file system watcher, and sets up the policy file watching.
// It returns a pointer to the created MiddlewareBuilder and any error encountered during the initialization process.
func InitMiddlewareBuilder(modelFile, policyFile string, subResolver SubResolver) (*MiddlewareBuilder, error) {
	// Create a new Casbin enforcer using the provided model and policy file paths.
	enforcer, err := casbin.NewEnforcer(modelFile, policyFile)
	if err != nil {
		return nil, err
	}

	// Initialize a new watcher to monitor the policy file for changes.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating file watcher: %v", err)
	}

	// Add the policy file to the watcher, so it starts monitoring for changes.
	err = watcher.Add(policyFile)
	if err != nil {
		return nil, fmt.Errorf("error adding policy file to watcher: %v", err)
	}

	// Return a pointer to the constructed MiddlewareBuilder with all its fields set accordingly.
	return &MiddlewareBuilder{
		enforcer:    enforcer,
		subResolver: subResolver,
		policyFile:  policyFile,
		watcher:     watcher,
	}, nil
}

// Close is a cleanup function to close the file watcher when the MiddlewareBuilder is no longer needed.
func (b *MiddlewareBuilder) Close() error {
	b.cacheMutex.Lock()         // Lock the mutex to ensure exclusive access while closing the watcher.
	defer b.cacheMutex.Unlock() // Ensure the mutex is unlocked after closing.
	return b.watcher.Close()    // Close the file watcher and return any errors.
}

// watchPolicyFile is a method that runs in its goroutine to listen for file events and update the policy when changes are detected.
func (b *MiddlewareBuilder) watchPolicyFile() {
	for {
		select {
		case event, ok := <-b.watcher.Events: // Receive file event notifications.
			if !ok {
				log.Println("file watcher channel closed")
				return
			}
			// If the event is a write operation, update the policy.
			if event.Op&fsnotify.Write == fsnotify.Write {
				err := b.UpdatePolicy()
				if err != nil {
					log.Printf("failed to load updated policy: %v", err)
				}
			}
		case err, ok := <-b.watcher.Errors: // Receive error notifications.
			if !ok {
				log.Println("file watcher error channel closed")
				return
			}
			log.Printf("file watcher error: %v", err)
		}
	}
}

// UpdatePolicy is a method that reloads the enforcement policy from the policy file.
func (b *MiddlewareBuilder) UpdatePolicy() error {
	b.cacheMutex.Lock()         // Lock the mutex for exclusive access to the policy.
	defer b.cacheMutex.Unlock() // Ensure the mutex is unlocked after the update.

	// Reload the policy from the policy file and return any errors encountered.
	return b.enforcer.LoadPolicy()
}

// Build is a method that constructs the middleware used to check permissions on each request.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	go b.watchPolicyFile() // Start the policy file watch in a separate goroutine.
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			defer func() {
				if r := recover(); r != nil {
					// Recover from a panic, if it happens, and log the error.
					log.Printf("recovered from panic in middleware: %v", r)
					sendError(ctx.ResponseWriter, http.StatusInternalServerError, "Internal Server Error")
				}
			}()
			// Use the subResolver to find out who the subject is from the HTTP request.
			sub, subErr := b.subResolver(ctx.Request)
			if subErr != nil {
				// If there is a problem resolving the subject, send an unauthorized error response.
				sendError(ctx.ResponseWriter, http.StatusUnauthorized, subErr.Error())
				return
			}

			// The requested object is the URL path of the HTTP request.
			obj := ctx.Request.URL.Path
			// The action is the HTTP method being used.
			act := ctx.Request.Method

			// Lock the mutex as a reader to allow multiple concurrent reads.
			b.cacheMutex.RLock()
			ok, enforceErr := b.enforcer.Enforce(sub, obj, act)
			// Unlock the mutex as soon as the read is done.
			b.cacheMutex.RUnlock()

			if enforceErr != nil {
				// If there's an issue with enforcement, send an internal error response.
				sendError(ctx.ResponseWriter, http.StatusInternalServerError, "Error checking permissions: "+enforceErr.Error())
				return
			}

			// If not ok, then permission is denied.
			if !ok {
				sendError(ctx.ResponseWriter, http.StatusForbidden, "Forbidden")
				return
			}

			// Log that permission was granted.
			log.Printf("Permission granted for sub: %s, obj: %s, act: %s\n", sub, obj, act)
			// Forward the request to the next middleware/handler.
			next(ctx)
		}
	}
}

// sendError is a helper function to send JSON formatted error responses.
func sendError(resp http.ResponseWriter, status int, message string) {
	resp.Header().Set("Content-Type", "application/json") // Set the Content-Type header to application/json.
	resp.WriteHeader(status)                              // Set the HTTP status code of the response.
	encodeErr := json.NewEncoder(resp).Encode(map[string]string{
		"error": message,
	})
	if encodeErr != nil {
		// If there's an error encoding the response, report it as an internal server error.
		http.Error(resp, fmt.Sprintf("error sending error response: %v", encodeErr), http.StatusInternalServerError)
	}
}
