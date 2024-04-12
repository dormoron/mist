package recovery

import "github.com/dormoron/mist"

// MiddlewareBuilder is a struct that encapsulates configuration options for creating HTTP
// middleware. The struct stores a status code for the HTTP response, a slice of bytes that can
// be used as response data, and a logging function that can be executed with a given context
// during the middleware's operation.
type MiddlewareBuilder struct {
	StatusCode int                     // The HTTP status code that the middleware will set on the response.
	Data       []byte                  // Data is a slice of bytes to be potentially used as the HTTP response body.
	Log        func(ctx *mist.Context) // Log is a function to execute logging operations within the middleware scope, receiving a context with relevant request information.
}

// Build is a method on the MiddlewareBuilder struct that constructs and returns a new middleware.
// The middleware created by this method is designed to intercept the processing of an HTTP
// request, allowing for actions to be taken before or after the next handler in the chain is called.
// The middleware also includes panic recovery, which sets a predefined response and logs the panic
// using the provided log function.
//
// Returns:
//   - mist.Middleware: This is the middleware function that conforms to the mist.Middleware type.
//     It takes the next handler in the chain and returns a new handler that performs
//     the middleware's operations.
//
// Example usage:
//
//	mb := MiddlewareBuilder{
//	    StatusCode: 500,
//	    Data:       []byte("Internal Server Error"),
//	    Log:        func(ctx *mist.Context) {
//	                   // Implementation of logging when a panic occurs
//	                },
//	}
//
// middleware := mb.Build()
// // This middleware can then be used to wrap around any mist.HandleFunc to apply the logic defined in Build.
func (m MiddlewareBuilder) Build() mist.Middleware {
	// Returns a new middleware closure that takes the next handler in the chain and returns a new handler.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Returns a new handler function that takes the request context.
		return func(ctx *mist.Context) {
			// Defer a function that recovers from any panics that happen down the call chain.
			defer func() {
				// Recover from panic if any have occurred.
				if r := recover(); r != nil {
					// Set the preconfigured response data and status code in the context,
					// to be returned to the client.
					ctx.RespData = m.Data
					ctx.RespStatusCode = m.StatusCode

					// Utilize the builder's Log function to log the panic context information.
					m.Log(ctx)
				}
			}()
			// Call the next handler in the chain with the updated context.
			next(ctx)
		}
	}
}
