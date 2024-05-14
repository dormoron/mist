package jwt

import (
	"fmt"
	"github.com/dormoron/mist"
	"github.com/golang-jwt/jwt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// MiddlewareOptions is a type that defines a function signature for functions that
// can be used to configure a MiddlewareBuilder. These functions receive a pointer
// to a MiddlewareBuilder instance and are used to modify its fields or set up configuration.
type MiddlewareOptions func(builder *MiddlewareBuilder)

// MiddlewareBuilder is a struct that holds configuration for creating
// a JWT (JSON Web Token) middleware.
type MiddlewareBuilder struct {
	StatusCode int                                 // StatusCode is the HTTP status code to return on error
	ErrMsg     string                              // ErrMsg is the error message to return to the client on failure
	LogFunc    func(ctx *mist.Context, msg string) // LogFunc is a logging function to record events
	Secret     []byte                              // Secret is the key used to validate the JWT
	Paths      []*regexp.Regexp                    // Paths is a slice of regular expressions that match paths to exclude from JWT checking
	IsHTTPS    bool
}

// InitMiddlewareBuilder creates and initializes a new MiddlewareBuilder object.
// secret: A byte slice used for signature validation of the token.
// statusCode: The HTTP status code to be used when returning errors.
// pathPatterns: A slice of strings representing the patterns that match paths that should skip JWT validation.
// It returns a new MiddlewareBuilder and error. If there's an error while compiling path patterns,
func InitMiddlewareBuilder(secret []byte, statusCode int, opts ...MiddlewareOptions) *MiddlewareBuilder {
	builder := &MiddlewareBuilder{
		Secret:     secret,
		StatusCode: statusCode,
		ErrMsg:     "Authentication Error",
		LogFunc:    defaultLogFunc,
		Paths:      make([]*regexp.Regexp, 0),
		IsHTTPS:    true,
	}
	for _, opt := range opts {
		opt(builder)
	}
	return builder
}

// WithLogFunc is a function that returns a MiddlewareOptions type. Its purpose is to provide
// a customization option for the logging function within a middleware setup. By using this
// function, developers can specify how logs should be handled or processed within the middleware.
// Parameters:
//   - logFunc: This is a function parameter that takes a custom logging function defined by the user.
//     The custom logging function must accept a context (of type *mist.Context) and a message
//     (of type string) as its parameters. The mist.Context parameter provides contextual information
//     about the incoming request, whereas the msg parameter is the message to be logged.
//
// Returns:
//   - A function of type MiddlewareOptions. MiddlewareOptions is a functional option type that allows
//     setting or modifying options/configurations of the MiddlewareBuilder. The returned function
//     takes a pointer to MiddlewareBuilder as its input and does not return anything. Its purpose is to
//     set the LogFunc field of the MiddlewareBuilder to the logFunc provided by the developer.
func WithLogFunc(logFunc func(ctx *mist.Context, msg string)) MiddlewareOptions {
	// Return a function that conforms to MiddlewareOptions. This returned function
	// will be used to modify a MiddlewareBuilder instance.
	return func(builder *MiddlewareBuilder) {
		// Assign the provided logFunc to the LogFunc field of the MiddlewareBuilder.
		// This allows the middleware to use the custom logging logic defined by the user.
		builder.LogFunc = logFunc
	}
}

// WithErrMsg is a function that returns MiddlewareOptions type. It provides a customization option
// for setting a custom error message within a MiddlewareBuilder instance. This custom error message
// can be used throughout the middleware processing pipeline for various purposes such as notifying
// developers about certain events or handling error conditions.
// Parameters:
// - errMsg: A string parameter provided by the developer that serves as a custom error message.
// Returns:
//   - A function of type MiddlewareOptions, which is essentially a function that modifies a MiddlewareBuilder
//     instance. The returned function takes a pointer to MiddlewareBuilder as its input and sets the ErrMsg
//     field of the MiddlewareBuilder to the errMsg provided by the developer.
func WithErrMsg(errMsg string) MiddlewareOptions {
	// Return a function that adheres to the MiddlewareOptions. This returned function will be used to modify
	// a MiddlewareBuilder instance.
	return func(builder *MiddlewareBuilder) {
		// Assign the provided errMsg to the ErrMsg field of the MiddlewareBuilder.
		// This lets the middleware use the custom error message set by the user in error handling.
		builder.ErrMsg = errMsg
	}
}

// WithHTTPS is a function that creates and returns a MiddlewareOptions function, which is used to configure
// the IsHTTPS field of a MiddlewareBuilder. This configuration function allows you to specify whether
// the middleware should enforce HTTPS connections.
//
// Parameters:
//   - isHTTPS: A boolean value that sets the middleware's IsHTTPS field.
//     If true, the middleware will enforce HTTPS connections.
//
// Returns:
//   - A MiddlewareOptions function that, when called with a MiddlewareBuilder, will set its IsHTTPS field
//     according to the value provided to WithHTTPS.
func WithHTTPS(isHTTPS bool) MiddlewareOptions {
	// The returned MiddlewareOptions function takes a pointer to a MiddlewareBuilder as its parameter.
	return func(builder *MiddlewareBuilder) {
		builder.IsHTTPS = isHTTPS // The IsHTTPS field of MiddlewareBuilder is set to the value passed to WithHTTPS.
	}
}

// WithPaths is a function that creates and returns a MiddlewareOptions function designed to configure
// the Paths field of a MiddlewareBuilder. This MiddlewareOptions function allows for specifying a list
// of string patterns representing the paths that the middleware should protect. Each provided string pattern
// is compiled into a regular expression.
//
// Parameters:
//   - pathPatterns: A slice of strings, where each string is a pattern representing a path to be protected
//     by the middleware. These patterns will be compiled into regular expressions.
//
// Returns:
//   - A MiddlewareOptions function that, when used with a MiddlewareBuilder, sets its Paths field with
//     the compiled regular expression patterns of the provided path patterns.
func WithPaths(pathPatterns []string) MiddlewareOptions {
	// The returned MiddlewareOptions function accepts a pointer to a MiddlewareBuilder.
	return func(builder *MiddlewareBuilder) {
		// Initialize a slice to store the compiled regular expressions. The capacity is set to the length
		// of the provided pathPatterns to optimize memory allocation.
		paths := make([]*regexp.Regexp, 0, len(pathPatterns))

		// Iterate over each pattern in the provided pathPatterns slice.
		for _, pattern := range pathPatterns {
			// Attempt to compile the current pattern into a regular expression.
			compiledPattern, err := regexp.Compile(pattern)
			if err != nil { // Check if there was an error during the compilation.
				// Log the error message and skip to the next iteration if the pattern fails to compile.
				log.Printf("failed to compile path pattern '%s': %v", pattern, err)
				continue
			}
			// If the pattern compiles successfully, append the compiled regular expression to the paths slice.
			paths = append(paths, compiledPattern)
		}

		// After processing all patterns, assign the resulting slice of compiled regular expressions
		// to the Paths field of the MiddlewareBuilder.
		builder.Paths = paths
	}
}

// defaultLogFunc is the default logging function used by the middleware.
// It logs a message with a timestamp to standard output.
func defaultLogFunc(ctx *mist.Context, msg string) {
	fmt.Printf("%s - %s\n", time.Now().Format(time.RFC3339), msg)
}

// Build constructs the middleware function that can be attached to a server.
// It involves token validation logic and error handling.
// next: The next HandleFunc in line to be executed after the middleware.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// Start off the token validation process by logging the initiation.
			m.LogFunc(ctx, "Starting the auth token validation")
			// If the MiddlewareBuilder was set to enforce HTTPS, and the request does not indicate a secure connection,
			// return a '401 Unauthorized' response.
			if m.IsHTTPS && ctx.Request.TLS == nil {
				m.sendError(ctx, http.StatusUnauthorized)
				return
			}
			// Check if the requested path is one of the paths that should skip the JWT check.
			for _, pattern := range m.Paths {
				if pattern.MatchString(ctx.Request.URL.Path) {
					next(ctx) // If a match is found, skip JWT check and proceed to the next handler.
					return
				}
			}
			// Validate the JWT.
			if err := m.validateToken(ctx); err != nil {
				// If validation fails, send an error response.
				m.sendError(ctx, m.StatusCode)
				return
			}
			// Log successful token validation.
			m.LogFunc(ctx, "Auth token validated successfully")
			// Proceed to the next handler.
			next(ctx)
		}
	}
}

// validateToken is a method of MiddlewareBuilder that validates the JSON Web Token (JWT) from the 'Authorization' header
// of an HTTP request. It handles the token extraction, parsing, and verification to ensure it is valid according to the
// secret key and signing method specified in the MiddlewareBuilder.
// Parameters:
// - ctx: A pointer to mist.Context, which provides contextual information about the incoming HTTP request.
// Returns:
//   - An error indicating the result of the token validation process. If the validation is successful, nil is returned. Otherwise,
//     an error is returned describing the nature of the issue with the token.
func (m *MiddlewareBuilder) validateToken(ctx *mist.Context) error {
	// Extract the 'Authorization' header from the incoming HTTP request.
	authHeader := ctx.Request.Header.Get("Authorization")

	// Check if the 'Authorization' header is empty. If so, log the error using the provided LogFunc and return an error.
	if authHeader == "" {
		m.LogFunc(ctx, "No auth token provided")
		return fmt.Errorf("missing auth token")
	}

	// Split the 'Authorization' header into parts to separate the token type and the token itself.
	parts := strings.Fields(authHeader)

	// Validate the format of the 'Authorization' header to confirm it consists of two parts and
	// the token type is 'Bearer'. If the format is incorrect, return an error.
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return fmt.Errorf("invalid auth header format")
	}

	// Assign the second part of the split header, which is the JWT, to tokenString.
	tokenString := parts[1]

	// Parse the JWT and validate its claims. The jwt.ParseWithClaims function is called with the tokenString, a reference
	// to StandardClaims to structure the parsed claims, and a callback function to handle the token validation logic.
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Check if the token's signing method matches the expected HMAC signing method.
		// If not, return an error specifying the unexpected signing method found in the token header.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key used to verify the token signature.
		return m.Secret, nil
	})

	// If error occurs during token parsing, log the error and return an error indicating the token is invalid.
	if err != nil {
		m.LogFunc(ctx, fmt.Sprintf("Invalid token: %v", err))
		return fmt.Errorf("invalid token")
	}

	// Convert the token.Claims to jwt.StandardClaims and check the token's validity and expiry.
	// If the token is not valid or has expired, return an error.
	if claims, ok := token.Claims.(*jwt.StandardClaims); !ok || !token.Valid || !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return fmt.Errorf("token is invalid or has expired")
	}

	// After successful validation, log the successful token validation and return nil.
	m.LogFunc(ctx, "Token validated successfully")
	return nil
}

// sendError is a method of MiddlewareBuilder that's responsible for constructing and sending an error response
// to the client whenever an error condition is encountered during the middleware's operation. It utilizes
// HTTP status codes and the standard 'WWW-Authenticate' header to convey the reason for the error in a format
// conforming to HTTP specifications.
// Parameters:
//   - ctx: A pointer to a mist.Context, which contains information about the HTTP request and response. It's
//     used here to manipulate the response that will be sent back to the client.
//   - statusCode: An integer representing the HTTP status code to be set in the error response. This helps
//     indicate the type of error that occurred (e.g., 401 for unauthorized access).
func (m *MiddlewareBuilder) sendError(ctx *mist.Context, statusCode int) {
	// Set the 'Content-Type' response header to 'application/json', indicating that the response body
	// will include a JSON object.
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")

	// Set the 'WWW-Authenticate' response header to provide authentication details. This is particularly
	// useful for scenarios where authentication fails. The header specifies 'Bearer' tokens are required and
	// hints that the realm is a staging site, thus suggesting the scope of the authentication.
	ctx.ResponseWriter.Header().Set("WWW-Authenticate", `Bearer realm="Access to the staging site", charset="UTF-8"`)

	// Write the specified HTTP status code to the response header. This informs the client about the
	// nature of the error encountered.
	ctx.ResponseWriter.WriteHeader(statusCode)

	// Construct a JSON string that includes the custom error message defined in the MiddlewareBuilder's
	// ErrMsg field. This message is intended to provide more specific information about the error.
	responseMsg := `{"error": "` + m.ErrMsg + `"}`

	// Write the constructed JSON error message to the response body. This completes the error response,
	// which is then sent to the client.
	_, _ = ctx.ResponseWriter.Write([]byte(responseMsg))

	// Log the error message using the middleware's LogFunc. This ensures that the error is recorded,
	// allowing for debugging or monitoring errors encountered during operation.
	m.LogFunc(ctx, fmt.Sprintf("Error: %s", m.ErrMsg))
}
