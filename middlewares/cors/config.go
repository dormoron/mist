package cors

import (
	"github.com/dormoron/mist"
	"net/http"
	"strings"
)

type cors struct {
	allowAllOrigins            bool                             // Whether to allow all origins.
	allowCredentials           bool                             // Whether to allow credentials in CORS requests.
	allowOriginFunc            func(string) bool                // Function to validate allowed origins based on string input.
	allowOriginWithContextFunc func(*mist.Context, string) bool // Function to validate allowed origins with context.
	allowOrigins               []string                         // List of explicitly allowed origins.
	normalHeaders              http.Header                      // Headers for normal (non-preflight) CORS requests.
	preflightHeaders           http.Header                      // Headers for preflight CORS requests.
	wildcardOrigins            [][]string                       // List of allowed wildcard origins.
	optionsResponseStatusCode  int                              // Response status code for preflight responses.
}

// Predefined schema lists for different types of URIs.
var (
	DefaultSchemas = []string{
		"http://",
		"https://",
	}
	ExtensionSchemas = []string{
		"chrome-extension://",
		"safari-extension://",
		"moz-extension://",
		"ms-browser-extension://",
	}
	FileSchemas = []string{
		"file://",
	}
	WebSocketSchemas = []string{
		"ws://",
		"wss://",
	}
)

// newCors creates a new cors struct based on the provided configuration.
// It validates configurations and initializes the struct fields accordingly.
//
// Parameters:
// - config: Configuration for the CORS settings.
//
// Returns:
// - A pointer to a newly created cors instance.
func newCors(config Config) *cors {
	// Validate the provided configuration.
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}

	// Check for wildcard origin indication.
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			config.AllowAllOrigins = true
		}
	}

	// Set the default preflight response status code if not provided.
	if config.OptionsResponseStatusCode == 0 {
		config.OptionsResponseStatusCode = http.StatusNoContent
	}

	// Create and return a new cors instance.
	return &cors{
		allowOriginFunc:            config.AllowOriginFunc,
		allowOriginWithContextFunc: config.AllowOriginWithContextFunc,
		allowAllOrigins:            config.AllowAllOrigins,
		allowCredentials:           config.AllowCredentials,
		allowOrigins:               normalize(config.AllowOrigins),
		normalHeaders:              generateNormalHeaders(config),
		preflightHeaders:           generatePreflightHeaders(config),
		wildcardOrigins:            config.parseWildcardRules(),
		optionsResponseStatusCode:  config.OptionsResponseStatusCode,
	}
}

// applyCors applies CORS settings to the incoming request based on the origin and method.
//
// Parameters:
// - c: The context of the current HTTP request.
func (cors *cors) applyCors(c *mist.Context) {
	origin := c.Request.Header.Get("Origin") // Extract the origin from the request headers.

	if len(origin) == 0 {
		// If the request is not a CORS request, do nothing.
		return
	}
	host := c.Request.Host

	if origin == "http://"+host || origin == "https://"+host {
		// If the origin matches the host, it is not a CORS request but has the origin header.
		// This can happen with the fetch API.
		return
	}

	// Check if the origin is valid.
	if !cors.isOriginValid(c, origin) {
		c.AbortWithStatus(http.StatusForbidden) // Abort the request with a 403 Forbidden status.
		return
	}

	// Handle preflight requests (OPTIONS method).
	if c.Request.Method == "OPTIONS" {
		cors.handlePreflight(c)
		defer c.AbortWithStatus(cors.optionsResponseStatusCode) // Defer abort with preflight response status.
	} else {
		// Handle normal CORS requests.
		cors.handleNormal(c)
	}

	if !cors.allowAllOrigins {
		c.Header("Access-Control-Allow-Origin", origin) // Set the allowed origin header.
	}
}

// validateWildcardOrigin checks if the given origin matches any of the wildcard rules.
//
// Parameters:
// - origin: The origin to validate.
//
// Returns:
// - A boolean indicating whether the origin is valid based on wildcard rules.
func (cors *cors) validateWildcardOrigin(origin string) bool {
	for _, w := range cors.wildcardOrigins {
		// Check if the origin matches any of the wildcard rules.
		if w[0] == "*" && strings.HasSuffix(origin, w[1]) {
			return true
		}
		if w[1] == "*" && strings.HasPrefix(origin, w[0]) {
			return true
		}
		if strings.HasPrefix(origin, w[0]) && strings.HasSuffix(origin, w[1]) {
			return true
		}
	}

	return false
}

// isOriginValid checks if the given origin is valid based on the configuration and context.
//
// Parameters:
// - c: The context of the current HTTP request.
// - origin: The origin to validate.
//
// Returns:
// - A boolean indicating whether the origin is valid.
func (cors *cors) isOriginValid(c *mist.Context, origin string) bool {
	valid := cors.validateOrigin(origin) // Validate the origin using the basic validation function.
	if !valid && cors.allowOriginWithContextFunc != nil {
		// If not valid by basic rules, use the context-aware validation function if provided.
		valid = cors.allowOriginWithContextFunc(c, origin)
	}
	return valid
}

// validateOrigin checks if the given origin is valid based on the configuration.
//
// Parameters:
// - origin: The origin to validate.
//
// Returns:
// - A boolean indicating whether the origin is valid.
func (cors *cors) validateOrigin(origin string) bool {
	if cors.allowAllOrigins {
		return true // Allow all origins if configured to do so.
	}
	for _, value := range cors.allowOrigins {
		if value == origin {
			return true // Direct match with allowed origins.
		}
	}
	// Check if the origin matches any of the wildcard rules.
	if len(cors.wildcardOrigins) > 0 && cors.validateWildcardOrigin(origin) {
		return true
	}
	// Validate using the custom origin validation function if provided.
	if cors.allowOriginFunc != nil {
		return cors.allowOriginFunc(origin)
	}
	return false
}

// handlePreflight handles CORS preflight requests by setting the appropriate headers.
//
// Parameters:
// - c: The context of the current HTTP request.
func (cors *cors) handlePreflight(c *mist.Context) {
	header := c.ResponseWriter.Header() // Get the response headers.
	for key, value := range cors.preflightHeaders {
		header[key] = value // Set the preflight headers.
	}
}

// handleNormal handles normal CORS requests by setting the appropriate headers.
//
// Parameters:
// - c: The context of the current HTTP request.
func (cors *cors) handleNormal(c *mist.Context) {
	header := c.ResponseWriter.Header() // Get the response headers.
	for key, value := range cors.normalHeaders {
		header[key] = value // Set the normal headers.
	}
}
