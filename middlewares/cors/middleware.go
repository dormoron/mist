package cors

import (
	"github.com/dormoron/mist"
	"net/http"
)

// MiddlewareBuilder is a struct that serves as a configuration holder for creating middleware, specifically tailored for managing
// Cross-Origin Resource Sharing (CORS) settings in HTTP handlers or servers. Middleware created using this builder can be used
// to dictate which origins are allowed to access resources on the server.
// Fields:
//   - AllowOrigin: A string that specifies the origin(s) that are permitted to access the resources. This can be a specific URI,
//     or "*" to allow any origin. It translates directly to the 'Access-Control-Allow-Origin' header in CORS
//     implementations. Correct configuration ensures that resources are accessible only to certain origins, thus
//     following the security practices pertaining to CORS.
//
// Usage of the AllowOrigin field aligns with web security principles inhibiting resources from being accessed by unauthorized origins.
// Proper setting of this field is crucial to safeguard against Cross-Site Scripting (XSS) and Cross-Site Request Forgery (CSRF)
// attacks by ensuring that only designated origins can make cross-origin HTTP requests to the server.
// Usage Example:
// To create middleware with this builder that will only allow requests from 'https://example.com', one would initialize a
// MiddlewareBuilder instance like this:
//
// builder := MiddlewareBuilder{AllowOrigin: "https://example.com"}
//
// The builder can then be used to create middleware for an HTTP server that will add the 'Access-Control-Allow-Origin' header
// with the value "https://example.com" to each response, thus allowing cross-origin requests exclusively from 'https://example.com'.
type MiddlewareBuilder struct {
	AllowOrigin string // URI(s) that are permitted to access the server
}

// Build constructs a middleware function that applies Cross-Origin Resource Sharing (CORS) headers
// to HTTP responses, based on the configuration set in the MiddlewareBuilder. This middleware can be
// attached to an HTTP server or router to ensure that CORS policies are enforced for every incoming request.
//
// Returns:
// - The middleware function that encapsulates the CORS logic.
//
// The middleware performs the following actions:
// 1. It resolves the `AllowOrigin` value to set the 'Access-Control-Allow-Origin' header.
//
//   - If the MiddlewareBuilder's `AllowOrigin` is set, it uses that value.
//
//   - Otherwise, it defaults to the origin specified in the request header.
//
//     2. It sets the 'Access-Control-Allow-Credentials' header to "true" to allow the client to send credentials
//     with cross-origin requests.
//
//     3. If the 'Access-Control-Allow-Headers' header is not already set, it adds 'Content-Type' as a permissible
//     header. This is a common header that needs to be accepted during CORS requests.
//
//     4. The middleware handles preflight (OPTIONS) requests by setting a 200 OK response and ends processing.
//     Preflight requests do not require further handling by other middleware or handlers.
//
// 5. For all other request methods, it calls the next handler in the chain.
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
