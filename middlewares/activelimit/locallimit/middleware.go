package locallimit

import (
	"github.com/dormoron/mist"
	"go.uber.org/atomic"
	"net/http"
)

// MiddlewareBuilder is a struct that is designed to build middleware for managing
// concurrency within your application. It is particularly useful for creating
// rate limiting middleware that limits the number of simultaneous requests that
// can be processed by the application at any given time, providing a means to
// handle overload scenarios gracefully.
// Fields:
//
//   - maxActive:
//     A pointer to an atomic int64 value representing the maximum number of active
//     requests allowed. The atomic package is used here to allow for lock-free,
//     thread-safe manipulation of the counter value which is crucial for maintaining
//     an accurate count in a concurrent environment.
//
//   - countActive:
//     A pointer to an atomic int64 that is used to keep track of the current number of
//     active requests being processed by the middleware. This value is incremented as
//     new requests come in and decremented as requests are finished or rejected. Just
//     like maxActive, atomic is used to ensure that these operations are safe to use
//     across multiple goroutines without risking race conditions or inaccuracies.
//
//   - overloadResponseHandler:
//     A function of type that takes *mist.Context as a
//     parameter. This function is called when a request exceeds the maxActive limit, and
//     it handles the response that should be returned in such overload scenarios. Having
//     this as a configurable field allows users to define custom behavior for
//     how they want to communicate with the client when too many requests are encountered,
//     such as sending a custom error response, logging the incident, or invoking other
//     monitoring mechanisms.
//
// This struct is the backbone of a concurrency management middleware system, enabling
// precise control over request flow and providing scalable solutions for web applications
// that may experience varying levels of traffic.
type MiddlewareBuilder struct {
	maxActive               *atomic.Int64
	countActive             *atomic.Int64
	overloadResponseHandler func(ctx *mist.Context)
}

// InitMiddlewareBuilder initializes a new MiddlewareBuilder with the specified maximum number of active requests.
// This constructor sets up the MiddlewareBuilder struct by initializing the maxActive and countActive atomic values,
// and setting the overloadResponseHandler to nil, indicating that there is no custom overload response logic provided by default.
// Parameters:
//   - maxActive: The maximum number of simultaneous active requests that the middleware will allow.
//     Requests beyond this count will either be rejected or handled using a specified overload response.
//
// Returns:
// - A pointer to the newly created MiddlewareBuilder instance.
// The function ensures that the core variables for rate limiting logic are thread-safe by using atomic
// variables, which allows for concurrent access without the need for locks, thus avoiding potential performance bottlenecks.
// Usage:
// Initialize a MiddlewareBuilder before attaching it to your web server or framework,
// allowing the builder to then generate middleware that will enforce the rate limiting.
func InitMiddlewareBuilder(maxActive int64) *MiddlewareBuilder {
	// Create a new MiddlewareBuilder instance with initialized values.
	return &MiddlewareBuilder{
		// maxActive is set to the value passed into the function, encapsulated in an atomic variable
		// for safe concurrent operations.
		maxActive: atomic.NewInt64(maxActive),

		// countActive is initialized to 0, as no requests are active at the start. This is also encapsulated
		// in an atomic variable for the same reasons as maxActive.
		countActive: atomic.NewInt64(0),

		// overloadResponseHandler is set to nil, meaning no custom logic is set for handling cases
		// where the maximum active request limit has been reached. It can be set to a non-nil value
		// later to handle such cases as required.
		overloadResponseHandler: nil,
	}
}

// SetOverloadResponseHandler sets a custom handler function that will be called whenever the MiddlewareBuilder
// detects that the number of active requests has exceeded the maximum allowed (maxActive).
// This method is a part of the MiddlewareBuilder and allows you to define how the system should respond
// when it is too busy to handle additional requests, providing a way to customize the behavior instead
// of using a default response such as an HTTP 429 Too Many Requests status code.
// Parameters:
//   - overloadResponseHandler: A function that takes a *mist.Context as its argument and does not return anything.
//     This function is executed when there's an attempt to process a request but the system
//     is already handling the maximum number of in-flight requests.
//
// Returns:
//   - The reference to the MiddlewareBuilder itself (*MiddlewareBuilder), allowing for method chaining when
//     configuring the builder in a fluent interface style.
//
// By setting a custom overload response handler, developers can introduce custom logic, such as logging details
// about the overload, informing the client with more specific error messages, introducing a retry-after header,
// or potentially queuing the request for later processing.
// Example:
// builder := InitMiddlewareBuilder(100) // Initialize MiddlewareBuilder with a maxActive of 100.
//
//	builder.SetOverloadResponseHandler(func(ctx *mist.Context) {
//	    // Custom logic to execute when an overload condition is met.
//	    ctx.AbortWithStatusJSON(http.StatusTooManyRequests, "Please retry after some time")
//	})
func (m *MiddlewareBuilder) SetOverloadResponseHandler(overloadResponseHandler func(ctx *mist.Context)) *MiddlewareBuilder {
	// Assign the custom handler function to the builder's overloadResponseHandler field.
	m.overloadResponseHandler = overloadResponseHandler

	// Return the MiddlewareBuilder instance to support method chaining.
	return m
}

// Build is a method on the MiddlewareBuilder struct that creates and returns a middleware function.
// This middleware function manages the concurrency of incoming requests based on the configurations
// stored in the MiddlewareBuilder (max active requests, current active requests, overload response handler).
// The returned middleware intercepts each incoming request, checks the current number of active requests
// against the maxActive value set in the MiddlewareBuilder. If the current active requests exceed maxActive,
// it either invokes a user-defined overload response handler (if one has been set) or returns an HTTP
// 'Too Many Requests' (status code 429) response to the client.
// If the maxActive limit hasn't been reached, it calls the next middleware function in the chain, allowing
// the request to be processed further.
// Returns:
// - The middleware function that manages the rate of incoming requests based on the set configurations.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// return a middleware function
	return func(next mist.HandleFunc) mist.HandleFunc {
		// return a new function that wraps the 'next' function
		return func(ctx *mist.Context) {

			// Add the new request to the count of active requests.
			current := m.countActive.Add(1)

			// Ensure that the count of active requests is decremented after the function completes.
			defer func() {
				m.countActive.Sub(1)
			}()

			// If the number of current active requests is greater than the maximum allowed,
			if current > m.maxActive.Load() {
				// Check if a custom overload response handler has been set,
				if m.overloadResponseHandler != nil {
					// If it has, invoke the custom response handler.
					m.overloadResponseHandler(ctx)
				} else {
					// If not, abort the request and respond with a 'Too Many Requests' HTTP status code.
					ctx.AbortWithStatus(http.StatusTooManyRequests)
				}
				// Exit the function, not processing the request further.
				return
			}
			// If the maximum number of active requests has not been exceeded, invoke the 'next' function,
			// allowing the request to be processed further.
			next(ctx)
		}
	}
}
