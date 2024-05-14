package accesslog

import (
	"encoding/json"
	"github.com/dormoron/mist"
	"log"
)

// MiddlewareBuilder is a struct that facilitates the creation of middleware functions with
// logging capabilities for a web service framework (hypothetical 'mist' framework in this context).
// It contains a field that holds a reference to a logging function which can be used to log messages.
// This struct could potentially be extended with additional fields and methods to further
// customize the behavior of the middleware being built, especially for handling logging in a
// consistent manner throughout the application.
//
// Fields:
//
//	logFunc: A function of type 'func(log string)' that represents the logging strategy
//	         the middleware will utilize. The 'logFunc' is intended to be called within middleware
//	         handlers to log various messages, like errors or informational messages, to a log output,
//	         such as the console, a file, or a remote log management system.
//
// Example of adding a log function to MiddlewareBuilder:
//
//	mb := &MiddlewareBuilder{
//	    logFunc: func(log string) {
//	        // Implementation of the log function, could log to stdout or a file, etc.
//	        fmt.Println("Log:", log)
//	    },
//	}
//
//	// With the logFunc set, MiddlewareBuilder can now be used to construct middleware that logs messages.
//	middleware := mb.Build() // Assuming Build method is implemented on MiddlewareBuilder.
//	// The middleware can be used within the web framework to apply logging per request.
type MiddlewareBuilder struct {
	// logFunc is a field that holds a user-defined function intended to handle logging of messages.
	// The function takes a single string argument which contains the log message to be recorded.
	// The behavior of logging—where and how the log messages are output—is determined by the implementation
	// of this function provided by the user.
	logFunc func(log string)
}

// LogFunc assigns a custom logging function to the MiddlewareBuilder instance. This method is used
// to define how messages should be logged when the middleware operates within the request handling flow.
// It accepts a function parameter that matches the signature needed for logging operations in the middleware.
// The method enables a fluent interface by returning the MiddlewareBuilder instance itself, allowing for
// method chaining when configuring the builder.
//
// Parameters:
//
//	fn: A function that takes a single string argument representing the log message. The function should
//	    contain the code that dictates how the log messages will be processed and where they will be sent.
//	    This could involve logging to the console, writing to a file, or sending the data to a log aggregation
//	    service depending on the actual implementation passed to this method.
//
// Returns:
//
//	*MiddlewareBuilder: A pointer to the current instance of the MiddlewareBuilder, allowing for additional
//	                     configuration calls to be chained.
//
// Example usage:
//
//	mb := new(MiddlewareBuilder).
//	        LogFunc(func(log string) {
//	            // Custom logging implementation, such as sending log to an external service.
//	            SendLogToService(log)
//	        })
//	// At this point, 'mb' has a logging function that sends logs to an external service.
func (b *MiddlewareBuilder) LogFunc(fn func(log string)) *MiddlewareBuilder {
	// Assigns the given function 'fn' as the logging function for MiddlewareBuilder instance 'b'.
	// This function will be used for logging within the middleware built using this builder.
	b.logFunc = fn

	// Returns the MiddlewareBuilder instance to allow for method chaining.
	return b
}

// InitMiddleware initializes a new instance of the MiddlewareBuilder struct with default
// configuration settings. It sets up a standard logging function that will log access
// events using the Go standard library's log package. The returned MiddlewareBuilder
// can be further configured with additional options before being used to create
// middleware for an HTTP server.
//
// Returns:
// - A pointer to a new MiddlewareBuilder with default log function configuration.
//
// Usage:
//   - builder := InitBuilder()
//     This will create a new MiddlewareBuilder with a default log function.
func InitMiddleware() *MiddlewareBuilder {
	// Create a new MiddlewareBuilder instance with default configurations.
	return &MiddlewareBuilder{
		// Set the logFunc field to a default function that uses the log package's Println method.
		// This function prints the provided accessLog string to the standard logger, which by default
		// outputs to os.Stderr. The log output includes a timestamp and the file name and line number
		// of the log call, a behavior determined by the log package's standard flags.
		logFunc: func(accessLog string) {
			log.Println(accessLog)
		},
	}
}

// Build constructs a middleware function that is compliant with the mist framework's Middleware type.
// The middleware created by this method encompasses a logging feature as configured via the MiddlewareBuilder.
// The middleware function created here, when executed, performs the following operations:
//   - It first creates a deferred function that collects access log data upon request completion and logs it
//     using the pre-configured `logFunc`.
//   - Then it continues with the execution of the next middleware or final request handler (`next`) in the stack.
//
// The method returns a closure that matches the mist.Middleware type signature and can be integrated into
// the middleware chain during the configuration of the HTTP server or router.
//
// Returns:
//   - mist.Middleware: A middleware function that wraps around the next handler in the middleware stack,
//     providing access logging functionalities.
//
// Example usage:
//   - After creating an instance of MiddlewareBuilder and setting the log function,
//     the Build method is used to obtain a new Middleware function that logs requests.
//     router.Use(mb.Build()) // Assuming 'router' is an instance that has a 'Use' method accepting Middleware.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		// The function returned here matches the mist.HandleFunc signature. It receives the next handler
		// in the middleware chain and also returns a mist.HandleFunc. This allows it to be used within
		// the 'mist' framework as a middleware.
		return func(ctx *mist.Context) {
			// Define a deferred function that will always run after the request processing is completed.
			// This deferred function creates an access log struct containing relevant request information,
			// marshals it to JSON, and then logs it using the `logFunc` defined in the MiddlewareBuilder.
			defer func() {
				// Compile access log information into a struct from the provided context `ctx`.
				log := accessLog{
					Host:       ctx.Request.Host,     // Hostname from the HTTP request
					Route:      ctx.MatchedRoute,     // The route pattern matched for the request
					HTTPMethod: ctx.Request.Method,   // HTTP method, e.g., GET, POST
					Path:       ctx.Request.URL.Path, // Request path
				}
				// Convert the access log struct to JSON format.
				data, _ := json.Marshal(log)
				// Log the access log JSON string via the logging function provided to the builder.
				// This employs the strategy we previously set with MiddlewareBuilder.LogFunc.
				b.logFunc(string(data))
			}()

			// Call the next handler in the middleware chain with the current context.
			next(ctx)
		}
	}
}

// accessLog defines the structure for logging HTTP request details. It is used within middleware to capture
// and marshal HTTP request information into a JSON format that can be logged by the configured log function.
// Each field in the struct is tagged with json struct tags to specify the JSON key names and omit the fields if
// they are empty when marshalling the struct into JSON format.
//
// Fields:
//
//	Host:       The domain name or IP address of the server that received the request.
//	            It is included in the HTTP request header and is represented here as a string.
//	            The `json:"host,omitempty"` struct tag indicates that this field will be marshalled into JSON
//	            with the key "host" and will be omitted from the marshalled JSON object if the field is empty.
//
//	Route:      The matched route pattern for the request. It provides context about which route was matched
//	            for a given request path, aiding in debugging and analytics.
//	            Similar to Host, it will only appear in JSON if it is not empty.
//
//	HTTPMethod: The HTTP method used for the request, e.g., GET, POST, PUT, DELETE, etc.
//	            This helps to understand the action that the client intended to perform.
//	            It will appear in the JSON with the key "http_method" if not empty.
//
//	Path:       The path of the request URL. This represents the specific endpoint or resource requested by the client.
//	            Documented in the JSON with the key "path" and is omitted if it is empty.
//
// An instance of accessLog is created and populated with data from an HTTP request context and then marshalled into JSON.
// The JSON output is then passed to a logging function to record the incoming requests being handled by an HTTP server.
//
// Example of accessLog JSON representation:
//
//	{
//	    "host": "example.com",
//	    "route": "/users/{userID}",
//	    "http_method": "GET",
//	    "path": "/users/123"
//	}
type accessLog struct {
	Host       string `json:"host,omitempty"`        // The server host name or IP address from the HTTP request.
	Route      string `json:"route,omitempty"`       // The matched route pattern for the request.
	HTTPMethod string `json:"http_method,omitempty"` // The HTTP method used in the request (e.g., GET, POST).
	Path       string `json:"path,omitempty"`        // The path of the HTTP request URL.
}
