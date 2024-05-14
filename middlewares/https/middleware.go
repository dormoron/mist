package https

import (
	"fmt"
	"github.com/dormoron/mist"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// RedirectConfig is a struct that encapsulates configurations for managing HTTP redirections.
// It is designed to enhance security and control over redirection behavior in web applications,
// allowing for fine-tuned adjustments to HTTP headers related to security.
// Fields:
//   - Enabled: A boolean flag that determines whether redirect functionality is enabled or not.
//     When set to true, redirection rules defined in this configuration will be applied.
//     This allows for enabling or disabling of redirects globally without removing the
//     configured settings.
//   - HSTSMaxAge: An integer that specifies the duration, in seconds, that the browser should
//     remember that this site is only to be accessed using HTTPS. This is a part of HTTP Strict
//     Transport Security (HSTS) policy, which helps protect against man-in-the-middle attacks
//     by forcing browsers to only use secure connections.
//   - CSP: A string that represents the Content-Security-Policy header value. This policy helps
//     to prevent a wide range of attacks including Cross-Site Scripting (XSS) and data
//     injection attacks by specifying valid sources of content.
//   - IncludeSubDomains: A boolean flag that, when set to true, applies the HSTS policy not only
//     to the domain but also to all of its subdomains. This ensures that the entire domain
//     hierarchy is only accessible over HTTPS.
//   - PreloadHSTS: A boolean flag that indicates whether the domain should be included in the
//     HSTS preload list. Submission to the preload list means that browsers will preload the
//     site's HSTS configuration, forcing HTTPS for the domain before the first visit, without
//     needing to receive the HSTS header via HTTP first.
//
// Usage:
//   - An instance of RedirectConfig can be initialized directly with desired configurations to
//     control the behavior of HTTP redirections and related security headers in web applications.
//
// Examples:
//   - var config = RedirectConfig{
//     Enabled: true,
//     HSTSMaxAge: 31536000, // Set to one year.
//     CSP: "default-src 'self'; script-src 'self' https://apis.example.com",
//     IncludeSubDomains: true,
//     PreloadHSTS: false, // Not preloaded by default.
//     }
type RedirectConfig struct {
	Enabled           bool
	HSTSMaxAge        int
	CSP               string
	IncludeSubDomains bool
	PreloadHSTS       bool
}

// MiddlewareBuilder is a struct that encapsulates configurations for building a middleware.
// It is designed for creating middleware in the context of an HTTP server
// using the mist framework. It manages redirecting requests based on the provided RedirectConfig struct settings.
// Fields:
//   - Config: An instantiated object of RedirectConfig struct. It holds configurations related to
//     HTTP redirection settings. It provides security and control for HTTP redirections. This includes
//     enforcing HTTPS redirect rules defined by the HTTP Strict Transport Security (HSTS) policy,
//     and setting Content-Security-Policy (CSP) headers to mitigate Cross-Site Scripting (XSS) and
//     other code injection attacks.
//
// Usage:
//   - An instance of MiddlewareBuilder can be initialized directly with a RedirectConfig,
//     applied when the middleware is performing redirections.
//
// Example:
//   - var builder = MiddlewareBuilder{
//     Config: RedirectConfig{
//     ...
//     },
//     }
type MiddlewareBuilder struct {
	Config RedirectConfig
}

// Build is a method of the MiddlewareBuilder struct. It constructs a new middleware function that
// will be composed into the request handling pipeline of an HTTP server. This middleware will enforce
// the redirection policies and security headers as defined in the MiddlewareBuilder's RedirectConfig.
// The returned mist.Middleware is a higher-order function that takes an existing mist.HandleFunc
// (which represents the next handler in the server's middleware chain) and wraps it with the
// additional functionality provided by the middleware.
// Returns:
//   - A mist.Middleware function ready to be used within an HTTP server setup that uses the mist framework.
//
// Usage:
//   - The middleware created by this method will:
//   - Check if the redirection is enabled and if not, simply pass control to the next handler.
//   - If redirection is enabled, it will enforce HTTPS by checking the request's TLS state and
//     'X-Forwarded-Proto' header.
//   - If the request is not already using HTTPS, it will redirect the client to the equivalent
//     HTTPS URL.
//   - Set the Strict-Transport-Security header according to the RedirectConfig.
//   - Optionally, set the Content-Security-Policy header if it's configured.
//
// Example:
//   - builder := MiddlewareBuilder{...}
//   - middleware := builder.Build()
//   - http.Handle("/", middleware(originalHandler))
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// Return a new middleware function.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// The function that will act as the middleware.
		return func(ctx *mist.Context) {
			// Skip redirection if not enabled in configuration.
			if !m.Config.Enabled {
				next(ctx)
				return
			}

			// Get either the X-Forwarded-Proto header or the protocol used in the request.
			proto := strings.Split(ctx.Request.Header.Get("X-Forwarded-Proto"), ",")[0]

			// Determine the host for the redirect. Prefers the X-Forwarded-Host header, falling
			// back to the host from the request if the header is not present.
			host := ctx.Request.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = ctx.Request.Host
			}

			// Create the HTTPS URL by combining the host with the requested URI.
			httpsURL := fmt.Sprintf("https://%s%s", host, ctx.Request.URL.RequestURI())

			// Validate the created HTTPS URL.
			if _, err := url.Parse(httpsURL); err != nil {
				// Log and return an internal server error if the URL is invalid.
				log.Printf("error: Invalid URL: %v, error: %v", httpsURL, err)
				http.Error(ctx.ResponseWriter, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Redirect the client to the HTTPS URL if the current request is not secure.
			if ctx.Request.TLS == nil && !strings.EqualFold(proto, "https") {
				http.Redirect(ctx.ResponseWriter, ctx.Request, httpsURL, http.StatusTemporaryRedirect)
				return
			}

			// Construct the HSTS header value based on configuration.
			hstsValue := fmt.Sprintf("max-age=%d", m.Config.HSTSMaxAge)
			if m.Config.IncludeSubDomains {
				hstsValue += "; includeSubDomains"
			}
			if m.Config.PreloadHSTS {
				hstsValue += "; preload"
			}
			// Set the Strict-Transport-Security header on the response.
			ctx.ResponseWriter.Header().Set("Strict-Transport-Security", hstsValue)

			// Set the Content-Security-Policy header if a policy has been defined.
			if m.Config.CSP != "" {
				ctx.ResponseWriter.Header().Set("Content-Security-Policy", m.Config.CSP)
			}

			// Proceed with the next function in the middleware chain.
			next(ctx)
		}
	}
}
