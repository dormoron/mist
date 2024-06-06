package cors

import (
	"errors"
	"fmt"
	"github.com/dormoron/mist"
	"strings"
	"time"
)

// Config represents all available options for the middleware.
type Config struct {
	AllowAllOrigins bool

	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is []
	AllowOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. It takes the origin
	// as an argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowOrigins is ignored.
	AllowOriginFunc func(origin string) bool

	// Same as AllowOriginFunc except also receives the full request context.
	// This function should use the context as a read only source and not
	// have any side effects on the request, such as aborting or injecting
	// values on the request.
	AllowOriginWithContextFunc func(c *mist.Context, origin string) bool

	// AllowMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS)
	AllowMethods []string

	// AllowPrivateNetwork indicates whether the response should include allow private network header
	AllowPrivateNetwork bool

	// AllowHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	AllowHeaders []string

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool

	// ExposeHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposeHeaders []string

	// MaxAge indicates how long (with second-precision) the results of a preflight request
	// can be cached
	MaxAge time.Duration

	// Allows to add origins like http://some-domain/*, https://api.* or http://some.*.subdomain.com
	AllowWildcard bool

	// Allows usage of popular browser extensions schemas
	AllowBrowserExtensions bool

	// Allows to add custom schema like tauri://
	CustomSchemas []string

	// Allows usage of WebSocket protocol
	AllowWebSockets bool

	// Allows usage of file:// schema (dangerous!) use it only when you 100% sure it's needed
	AllowFiles bool

	// Allows to pass custom OPTIONS response status code for old browsers / clients
	OptionsResponseStatusCode int
}

// AddAllowMethods adds custom HTTP methods to the list of allowed methods in the CORS configuration.
//
// Parameters:
// - methods: A variadic list of strings representing the custom HTTP methods to be allowed.
//
// Returns:
// - None
//
// Usage:
// config := &Config{}
// config.AddAllowMethods("CUSTOM-METHOD")
func (c *Config) AddAllowMethods(methods ...string) {
	c.AllowMethods = append(c.AllowMethods, methods...)
}

// AddAllowHeaders adds custom headers to the list of allowed headers in the CORS configuration.
//
// Parameters:
// - headers: A variadic list of strings representing the custom headers to be allowed.
//
// Returns:
// - None
//
// Usage:
// config := &Config{}
// config.AddAllowHeaders("X-Custom-Header")
func (c *Config) AddAllowHeaders(headers ...string) {
	c.AllowHeaders = append(c.AllowHeaders, headers...)
}

// AddExposeHeaders adds custom headers to the list of headers exposed to the client.
// Expose headers define which headers can be accessed by the client.
//
// Parameters:
// - headers: A variadic list of strings representing the custom headers to be exposed.
//
// Returns:
// - None
//
// Usage:
// config := &Config{}
// config.AddExposeHeaders("X-Expose-Header")
func (c *Config) AddExposeHeaders(headers ...string) {
	c.ExposeHeaders = append(c.ExposeHeaders, headers...)
}

// getAllowedSchemas returns a list of allowed URI schemas based on the configuration settings.
// It combines the default schemas with any additional schemas allowed for browser extensions,
// WebSockets, files, and custom schemas provided in the configuration.
//
// Parameters:
// - None
//
// Returns:
// - A slice of strings representing the allowed URI schemas.
func (c *Config) getAllowedSchemas() []string {
	allowedSchemas := DefaultSchemas // Start with the default schemas.

	// If browser extensions are allowed, add extension schemas.
	if c.AllowBrowserExtensions {
		allowedSchemas = append(allowedSchemas, ExtensionSchemas...)
	}

	// If WebSockets are allowed, add WebSocket schemas.
	if c.AllowWebSockets {
		allowedSchemas = append(allowedSchemas, WebSocketSchemas...)
	}

	// If file URIs are allowed, add file schemas.
	if c.AllowFiles {
		allowedSchemas = append(allowedSchemas, FileSchemas...)
	}

	// If there are custom schemas provided, add them.
	if c.CustomSchemas != nil {
		allowedSchemas = append(allowedSchemas, c.CustomSchemas...)
	}

	return allowedSchemas
}

// validateAllowedSchemas checks if the given origin matches any of the allowed schemas
// as specified in the configuration.
//
// Parameters:
// - origin: The origin URI to validate.
//
// Returns:
// - A boolean indicating whether the origin is valid based on the allowed schemas.
func (c *Config) validateAllowedSchemas(origin string) bool {
	allowedSchemas := c.getAllowedSchemas() // Get the list of allowed schemas.

	// Check if the origin starts with any of the allowed schemas.
	for _, schema := range allowedSchemas {
		if strings.HasPrefix(origin, schema) {
			return true
		}
	}

	return false
}

// Validate checks the CORS configuration for any conflicting settings and validates the origins defined.
// If the configuration is invalid, it returns an error.
//
// Parameters:
// - None
//
// Returns:
// - An error if the configuration is invalid, otherwise nil.
func (c *Config) Validate() error {
	hasOriginFn := c.AllowOriginFunc != nil
	hasOriginFn = hasOriginFn || c.AllowOriginWithContextFunc != nil

	// Check for conflicts between allowing all origins and specifying individual origins or origin functions.
	if c.AllowAllOrigins && (hasOriginFn || len(c.AllowOrigins) > 0) {
		originFields := strings.Join([]string{
			"AllowOriginFunc",
			"AllowOriginFuncWithContext",
			"AllowOrigins",
		}, " or ")
		return fmt.Errorf(
			"conflict settings: all origins enabled. %s is not needed",
			originFields,
		)
	}

	// Ensure that if not allowing all origins, there must be some way to validate origins.
	if !c.AllowAllOrigins && !hasOriginFn && len(c.AllowOrigins) == 0 {
		return errors.New("conflict settings: all origins disabled")
	}

	// Validate each origin in the list of allowed origins.
	for _, origin := range c.AllowOrigins {
		if !strings.Contains(origin, "*") && !c.validateAllowedSchemas(origin) {
			return errors.New("bad origin: origins must contain '*' or include " + strings.Join(c.getAllowedSchemas(), ","))
		}
	}

	return nil
}

// parseWildcardRules processes the allowed origins to extract wildcard rules for origin validation.
// It returns a slice of string slices, each representing a wildcard rule.
//
// Parameters:
// - None
//
// Returns:
// - A slice of string slices representing the wildcard rules.
func (c *Config) parseWildcardRules() [][]string {
	var wRules [][]string // Initialize an empty slice for wildcard rules.

	// If wildcards are not allowed, return the empty slice.
	if !c.AllowWildcard {
		return wRules
	}

	// Process each allowed origin to extract wildcard rules.
	for _, o := range c.AllowOrigins {
		if !strings.Contains(o, "*") {
			continue
		}

		if c := strings.Count(o, "*"); c > 1 {
			panic(errors.New("only one * is allowed").Error())
		}

		i := strings.Index(o, "*")
		if i == 0 {
			wRules = append(wRules, []string{"*", o[1:]})
			continue
		}
		if i == (len(o) - 1) {
			wRules = append(wRules, []string{o[:i], "*"})
			continue
		}

		wRules = append(wRules, []string{o[:i], o[i+1:]})
	}

	return wRules
}

// DefaultConfig returns a default CORS configuration with standard allowed methods and headers,
// no credentials allowed, and a default max age of 12 hours.
//
// Parameters:
// - None
//
// Returns:
// - A Config struct with the default settings.
func DefaultConfig() Config {
	return Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// Default creates a default middleware with settings that allow all origins.
// It uses the default configuration and sets AllowAllOrigins to true.
//
// Parameters:
// - None
//
// Returns:
// - A mist.Middleware function that applies the default CORS settings.
func Default() mist.Middleware {
	config := DefaultConfig()     // Get the default configuration.
	config.AllowAllOrigins = true // Allow all origins.
	return New(config)            // Create and return the middleware.
}

// New creates a new middleware based on the provided configuration. It initializes a cors instance
// and returns a middleware function that applies CORS settings to incoming requests.
//
// Parameters:
// - config: The configuration for the CORS settings.
//
// Returns:
// - A mist.Middleware function that applies the CORS settings based on the provided configuration.
func New(config Config) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		cors := newCors(config) // Initialize a new cors instance.
		return func(c *mist.Context) {
			cors.applyCors(c) // Apply CORS settings to the context.
		}
	}
}
