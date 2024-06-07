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
			if allowOrigin == "" {
				allowOrigin = ctx.Request.Header.Get("Origin")
			}
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			// ctx.Resp.Header().Set("Access-Control-Allow-Origin", "*")
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
			if ctx.ResponseWriter.Header().Get("Access-Control-Allow-Headers") == "" {
				ctx.ResponseWriter.Header().Add("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			}
			if ctx.Request.Method == http.MethodOptions {
				ctx.RespStatusCode = 200
				ctx.RespData = []byte("ok")
				next(ctx)
				return
			}
			next(ctx)
		}
	}
}
