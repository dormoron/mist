package jwt

import (
	"fmt"
	"github.com/dormoron/mist"
	"github.com/golang-jwt/jwt"
	"regexp"
	"strings"
	"time"
)

// MiddlewareBuilder is a struct that holds configuration for creating
// a JWT (JSON Web Token) middleware.
type MiddlewareBuilder struct {
	StatusCode int                                 // StatusCode is the HTTP status code to return on error
	ErrMsg     string                              // ErrMsg is the error message to return to the client on failure
	LogFunc    func(ctx *mist.Context, msg string) // LogFunc is a logging function to record events
	Secret     []byte                              // Secret is the key used to validate the JWT
	Paths      []*regexp.Regexp                    // Paths is a slice of regular expressions that match paths to exclude from JWT checking
}

// NewMiddlewareBuilder creates and initializes a new MiddlewareBuilder object.
// secret: A byte slice used for signature validation of the token.
// statusCode: The HTTP status code to be used when returning errors.
// errMsg: Default error message returned to the client.
// logFunc: Function used for logging activities within the middleware.
// pathPatterns: A slice of strings representing the patterns that match paths that should skip JWT validation.
// It returns a new MiddlewareBuilder and error. If there's an error while compiling path patterns,
// it will return an error.
func NewMiddlewareBuilder(secret []byte, statusCode int, errMsg string, logFunc func(ctx *mist.Context, msg string), pathPatterns []string) (*MiddlewareBuilder, error) {
	if logFunc == nil {
		logFunc = defaultLogFunc // Use the defaultLogFunc if no logging function is provided
	}
	paths := make([]*regexp.Regexp, 0, len(pathPatterns))
	for _, pattern := range pathPatterns {
		compile, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile path pattern '%s': %w", pattern, err)
		}
		paths = append(paths, compile)
	}

	return &MiddlewareBuilder{
		Secret:     secret,
		StatusCode: statusCode,
		ErrMsg:     "Authentication Error",
		LogFunc:    logFunc,
		Paths:      paths,
	}, nil
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

// validateToken checks the 'Authorization' header for the 'Bearer' token,
// validates it, and returns an error if the token is missing, invalid, or expired.
func (m *MiddlewareBuilder) validateToken(ctx *mist.Context) error {
	// Retrieve the 'Authorization' header from the request.
	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		// If the 'Authorization' header is missing, log and return an error.
		m.LogFunc(ctx, "No auth token provided")
		return fmt.Errorf("missing auth token")
	}

	// Split the header into its components.
	parts := strings.Fields(authHeader)
	// Ensure the header is correctly formatted.
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return fmt.Errorf("invalid auth header format")
	}

	// The actual token is the second part of the header.
	tokenString := parts[1]
	// Parse the token and validate its signature using the provided secret key.
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token's signing algorithm matches the expected algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})

	// Handle any token parsing errors.
	if err != nil {
		// Log the detailed invalid token error.
		m.LogFunc(ctx, fmt.Sprintf("Invalid token: %v", err))
		return fmt.Errorf("invalid token")
	}

	// Validate the claims of the token to ensure it's still valid and hasn't expired.
	if claims, ok := token.Claims.(*jwt.StandardClaims); !ok || !token.Valid || !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return fmt.Errorf("token is invalid or has expired")
	}

	// If everything checks out, log success and return no error.
	m.LogFunc(ctx, "Token validated successfully")
	return nil
}

// sendError constructs a JSON formatted error response using the given status code
// and default error message provided when creating the MiddlewareBuilder.
// It also sets appropriate headers to indicate the nature of the authentication error.
func (m *MiddlewareBuilder) sendError(ctx *mist.Context, statusCode int) {
	// Set the 'Content-Type' header to indicate the type of response.
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	// Set the 'WWW-Authenticate' header to provide authentication instructions.
	ctx.ResponseWriter.Header().Set("WWW-Authenticate", `Bearer realm="Access to the staging site", charset="UTF-8"`)
	// Write the status code to the response.
	ctx.ResponseWriter.WriteHeader(statusCode)
	// Create and send the JSON formatted error message.
	responseMsg := `{"error": "` + m.ErrMsg + `"}`
	ctx.ResponseWriter.Write([]byte(responseMsg))
	// Log the error message that was sent to the client.
	m.LogFunc(ctx, fmt.Sprintf("Error: %s", m.ErrMsg))
}
