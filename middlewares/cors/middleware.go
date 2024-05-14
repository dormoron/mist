package cors

import (
	"github.com/dormoron/mist"
	"net/http"
)

type MiddlewareBuilder struct {
	AllowOrigin string // URI(s) that are permitted to access the server
}

func InitMiddlewareBuilder() *MiddlewareBuilder {
	builder := &MiddlewareBuilder{
		AllowOrigin: "",
	}
	return builder
}

func (m *MiddlewareBuilder) SetAllowOrigin(allowOrigin string) *MiddlewareBuilder {
	m.AllowOrigin = allowOrigin
	return m
}

func (m *MiddlewareBuilder) Build() mist.Middleware {
	// Define and return the middleware function.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Define the function that will be executed as middleware.
		return func(ctx *mist.Context) {
			// Determine the 'Access-Control-Allow-Origin' value.
			allowOrigin := m.AllowOrigin
			// If not set in MiddlewareBuilder, use the origin from the request.
			if allowOrigin == "" {
				allowOrigin = ctx.Request.Header.Get("Origin")
			}
			// Set the 'Access-Control-Allow-Origin' header in the response.
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", allowOrigin)

			// Set 'Access-Control-Allow-Credentials' to "true".
			ctx.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")

			// If the 'Access-Control-Allow-Headers' is not set, add 'Content-Type' to it.
			if ctx.ResponseWriter.Header().Get("Access-Control-Allow-Headers") == "" {
				ctx.ResponseWriter.Header().Add("Access-Control-Allow-Headers", "Content-Type")
			}

			// Handle preflight OPTIONS request.
			if ctx.Request.Method == http.MethodOptions {
				// Set the response status code to 200.
				ctx.RespStatusCode = 200
				// Send "ok" as the response data.
				ctx.RespData = []byte("ok")
				// End processing by returning early.
				return
			}

			// Call the next middleware/handler in the chain.
			next(ctx)
		}
	}
}
