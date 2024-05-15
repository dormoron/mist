package cors

import (
	"github.com/dormoron/mist"
	"net/http"
)

type MiddlewareBuilder struct {
	AllowOrigin string // URI(s) that are permitted to access the server.
}

// InitMiddlewareBuilder initializes a new MiddlewareBuilder instance with default settings.
// It sets the AllowOrigin field to an empty string, indicating no origin is explicitly allowed by default.
// Returns:
// - A pointer to the newly created MiddlewareBuilder instance.
func InitMiddlewareBuilder() *MiddlewareBuilder {
	builder := &MiddlewareBuilder{
		AllowOrigin: "", // Initialize AllowOrigin as empty, implying no specific origin is allowed by default.
	}
	return builder // Return the newly initialized MiddlewareBuilder instance.
}

// SetAllowOrigin sets the origin that is allowed by the middleware to access the server.
// This method can be used to configure Access-Control-Allow-Origin headers for CORS (Cross-Origin Resource Sharing) requests.
// Parameters:
// - allowOrigin: A string specifying the URI that is permitted to access the server.
// Returns:
// - A pointer to the MiddlewareBuilder instance to enable method chaining.
func (m *MiddlewareBuilder) SetAllowOrigin(allowOrigin string) *MiddlewareBuilder {
	m.AllowOrigin = allowOrigin // Set the provided origin as the allowed origin.
	return m                    // Return the MiddlewareBuilder instance for chaining.
}

// Build constructs and returns the middleware function configured by the MiddlewareBuilder instance.
// This function sets up CORS headers based on the configuration provided to the MiddlewareBuilder instance.
// Returns:
// - A function that conforms to mist.Middleware signature, capturing the logic for handling CORS requests.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// Define and return the middleware function.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Define the function that will be executed as middleware.
		return func(ctx *mist.Context) {
			// Determine the 'Access-Control-Allow-Origin' value based on the MiddlewareBuilder configuration.
			allowOrigin := m.AllowOrigin
			// If AllowOrigin is not set in MiddlewareBuilder, use the origin from the incoming request.
			if allowOrigin == "" {
				allowOrigin = ctx.Request.Header.Get("Origin")
			}
			// Set the 'Access-Control-Allow-Origin' header in the response to the determined value.
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", allowOrigin)

			// Set 'Access-Control-Allow-Credentials' header to "true" to allow credentials to be included in CORS requests.
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")

			// Set default 'Access-Control-Allow-Headers' to include 'Content-Type' if not already specified.
			if ctx.ResponseWriter.Header().Get("Access-Control-Allow-Headers") == "" {
				ctx.ResponseWriter.Header().Add("Access-Control-Allow-Headers", "Content-Type")
			}

			// Handle preflight OPTIONS requests specifically, allowing for CORS preflight checks.
			if ctx.Request.Method == http.MethodOptions {
				// Set the response status code to 200 to indicate success.
				ctx.RespStatusCode = 200
				// Include a simple "ok" response data.
				ctx.RespData = []byte("ok")
				// Exit early to complete processing of the OPTIONS request.
				return
			}

			// For non-OPTIONS requests, call the next middleware or handler in the chain.
			next(ctx)
		}
	}
}
