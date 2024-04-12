package recovery

import "github.com/dormoron/mist"

// MiddlewareBuilder is a struct that encapsulates configurations for building middleware.
// It is designed to be used for creating middleware that can handle errors, log requests,
// and manage HTTP response behavior in a flexible and customizable way within an HTTP server
// using the mist framework.
//
// Fields:
//
//   - StatusCode: An integer that represents the default HTTP status code to be sent out when
//     an error occurs within the middleware. This is a convenience for setting a
//     uniform error response code.
//
//   - ErrMsg: A byte slice that holds the default error message to be sent out with the response
//     when an error is encountered. This message is meant to be generic and not disclose
//     sensitive error details that might be exploited.
//
//   - LogFunc: A function that provides logging capabilities for the middleware. It accepts a
//     context object from the mist framework (commonly as `ctx`) which contains details
//     about the request, and an error object (`err`) that may have been captured during
//     the request's lifecycle. It's user-definable, so it can be tailored to log whatever
//     information is necessary in a particular format or to a particular logging sink.
//
// Usage:
//   - An instance of MiddlewareBuilder can be initialized directly with desired configurations,
//     or it may be set up via a constructor-like function which provides defaults that can then
//     be modified.
//
// Examples:
//   - var builder = MiddlewareBuilder{
//     StatusCode: 500, // Set a generic 500 Internal Server Error status code by default.
//     ErrMsg: []byte("An unexpected error occurred"), // Set a generic error message.
//     LogFunc: func(ctx *mist.Context, err any) {
//     // A custom log function that logs to the standard output including the error and request path.
//     fmt.Printf("Error: %v, Path: %s\n", err, ctx.Request.Path)
//     },
//     }
type MiddlewareBuilder struct {
	// StatusCode is the HTTP status code used when an error needs to be conveyed to the client.
	StatusCode int

	// ErrMsg is the content that will be sent in the HTTP response body when an error occurs.
	ErrMsg []byte

	// LogFunc is a callback function to be executed when logging is required. For example,
	// this function can be called upon an error to log the incident for monitoring or debugging.
	LogFunc func(ctx *mist.Context, err any)
}

// Build creates and returns a mist.Middleware based on the configurations provided in the MiddlewareBuilder.
// The returned middleware is responsible for recovering from panics that may occur in the HTTP request
// handling cycle, logging the error, and returning a specified error response to the client.
//
// Returns:
// - A configured middleware function that incorporates error recovery and logging as defined in MiddlewareBuilder.
//
// Usage:
//   - mw := builder.Build()
//     After calling Build on a MiddlewareBuilder instance, the resulting middleware can be applied to an HTTP route.
func (m MiddlewareBuilder) Build() mist.Middleware {
	// Construct and return the middleware function.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Return a new handler function encapsulating the middleware logic.
		return func(ctx *mist.Context) {
			// Use defer and recover to catch any panics that occur during the HTTP handling cycle.
			defer func() {
				if err := recover(); err != nil {
					// In case of panic, set the context response data and status code to the ones specified in MiddlewareBuilder.
					ctx.RespData = m.ErrMsg
					ctx.RespStatusCode = m.StatusCode
					// Use LogFunc to log the error along with context information.
					m.LogFunc(ctx, err)
				}
			}()
			// Call the next middleware/handler in the chain.
			next(ctx)
		}
	}
}
