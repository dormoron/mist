package basic

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/dormoron/mist"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// MiddlewareBuilder is a struct that holds the configuration options for constructing a middleware.
// This configuration will eventually be used to create an instance of a middleware that performs
// actions like auth checks on HTTP requests. The structure is set up in a way to be
// modified using MiddlewareOptions functions.
type MiddlewareBuilder struct {
	RequiredUserHash     []byte // Byte slice containing the hash of the user credential required for auth.
	RequiredPasswordHash []byte // Byte slice containing the hash of the password credential required for auth.
	Realm                string // The 'realm' is a string to be displayed to the user when auth is required.
	// It is part of the HTTP Basic Authentication standard, to give the user a context or
	// description of what area or resource the credentials will grant access to.
	Paths []*regexp.Regexp // A slice of pointer to regular expressions. Each pattern in this slice will be used
	// to match request paths that the middleware should protect. Only requests to paths
	// that match any of these patterns will be subject to auth checks.
	IsHTTPS bool // A boolean flag indicating whether the middleware should force the use of HTTPS.
	// If this flag is set to true, the middleware will only allow requests made over
	// HTTPS and will reject any HTTP connections.
}

// InitMiddlewareBuilder is a function that initializes a new MiddlewareBuilder with hashed credentials and other provided options.
// It takes the required username and password for auth, a realm for the auth prompt, and an optional list
// of MiddlewareOptions functions to further customize the builder.
//
// Parameters:
//   - requiredUser: The username that will be required for auth. It will be hashed and stored in the builder.
//   - requiredPassword: The password that will be required for auth. It will also be hashed and stored.
//   - realm: A description or name for the protected area used in the auth prompt to inform the user.
//
// Returns:
// - A pointer to an initialized MiddlewareBuilder containing the hashed credentials and applied options.
// - An error if hashing of the provided credentials fails.
func InitMiddlewareBuilder(requiredUser, requiredPassword, realm string) (*MiddlewareBuilder, error) {
	var requiredUserHash []byte // Declare a byte slice to store the hashed username.

	// Check if a username was provided and generate a hash for it using bcrypt.
	if requiredUser != "" {
		var err error // Declare an error variable.
		// GenerateFromPassword is a bcrypt function for hashing passwords (or in this case, the username).
		requiredUserHash, err = bcrypt.GenerateFromPassword([]byte(requiredUser), bcrypt.DefaultCost)
		if err != nil { // Check if there was an error during the hash generation.
			return nil, fmt.Errorf("error hashing user: %w", err) // Wrap and return the error if hashing failed.
		}
	}

	// Similarly, generate a hash for the required password.
	requiredPasswordHash, err := bcrypt.GenerateFromPassword([]byte(requiredPassword), bcrypt.DefaultCost)
	if err != nil { // Check if there was an error during the hash generation.
		return nil, fmt.Errorf("error hashing password: %w", err) // Wrap and return the error if hashing failed.
	}

	// Initialize the MiddlewareBuilder struct with the generated hashes, realm, and default values.
	builder := &MiddlewareBuilder{
		RequiredUserHash:     requiredUserHash,
		RequiredPasswordHash: requiredPasswordHash,
		Realm:                realm,
		Paths:                make([]*regexp.Regexp, 0), // An empty slice for path patterns.
		IsHTTPS:              true,                      // By default, HTTPS is enforced.
	}

	// Return a pointer to the initialized MiddlewareBuilder and nil for the error since all went well.
	return builder, nil
}

// RefuseHTTPS configures the MiddlewareBuilder to not require HTTPS for connections.
// This can be helpful in environments where HTTPS is not available or necessary.
// It sets the IsHTTPS field of MiddlewareBuilder to false.
// Returns:
// - the pointer to the MiddlewareBuilder instance to allow method chaining.
func (m *MiddlewareBuilder) RefuseHTTPS() *MiddlewareBuilder {
	m.IsHTTPS = false // Set the IsHTTPS flag to false.
	return m          // Return the MiddlewareBuilder instance for chaining.
}

// IgnorePaths compiles the provided path patterns into regular expressions and adds them to the MiddlewareBuilder.
// This method allows specifying which paths the middleware should apply to.
// Parameters:
// - pathPatterns: a slice of strings representing the path patterns to be added.
// Returns:
// - the pointer to the MiddlewareBuilder instance to allow method chaining.
func (m *MiddlewareBuilder) IgnorePaths(pathPatterns []string) *MiddlewareBuilder {
	// Initialize a slice to store the compiled regular expressions. The capacity is set to the length of pathPatterns for efficiency.
	paths := make([]*regexp.Regexp, 0, len(pathPatterns))

	for _, pattern := range pathPatterns {
		// Attempt to compile the current pattern into a regular expression.
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			// If there's an error during compilation, log it and skip adding this pattern.
			log.Printf("failed to compile path pattern '%s': %v", pattern, err)
			continue
		}
		// Add the successfully compiled pattern to the slice of regular expressions.
		paths = append(paths, compiledPattern)
	}
	// Update the MiddlewareBuilder's Paths field with the compiled patterns.
	m.Paths = paths
	return m // Return the MiddlewareBuilder instance for chaining.
}

// Build is a method of the MiddlewareBuilder struct that constructs a Middleware function
// that can be used to perform Basic Authentication on incoming HTTP requests.
// The function checks if a request's path matches any provided patterns, if the request was made using HTTPS,
// and if the provided auth credentials are valid before allowing the request to continue.
//
// Returns:
// - a Middleware function that performs Basic Authentication and can be used in a middleware chain in a mist server.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// The Middleware function accepts a mist.HandleFunc, and returns a mist.HandleFunc.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// The returned mist.HandleFunc takes a *mist.Context, which provides access to the request and response.
		return func(ctx *mist.Context) {
			// If the MiddlewareBuilder was set to enforce HTTPS, and the request does not indicate a secure connection,
			// return a '401 Unauthorized' response.
			if m.IsHTTPS && ctx.Request.TLS == nil {
				unauthorized(ctx.ResponseWriter, m.Realm)
				return
			}

			// Retrieve the path from the request URL.
			requestPath := ctx.Request.URL.Path

			// If the request's path matches any of the path patterns,
			// call the next mist.HandleFunc in the chain and return from the current one.
			for _, pattern := range m.Paths {
				if pattern.MatchString(requestPath) {
					next(ctx)
					return
				}
			}

			// Extract the Authorization header from the request,
			// parse it to get the provided username and password, and whether or not they were provided.
			givenUser, givenPassword, ok := parseBasicAuth(ctx.Request.Header.Get("Authorization"))

			// If no valid Authorization header was provided, return a '401 Unauthorized' response.
			if !ok {
				unauthorized(ctx.ResponseWriter, m.Realm)
				return
			}

			// If the provided credentials do not match the required credentials,
			// return a '401 Unauthorized' response.
			if !checkCredentials(m, givenUser, givenPassword) {
				unauthorized(ctx.ResponseWriter, m.Realm)
				return
			}

			// If everything checks out, the request is authenticated and we call the next mist.HandleFunc in the chain.
			next(ctx)
		}
	}
}

// checkCredentials is a helper function for validating user credentials.
// It compares the provided username and password with the required credentials
// stored in the MiddlewareBuilder, using bcrypt's CompareHashAndPassword function.
//
// Parameters:
//   - m: A pointer to MiddlewareBuilder which contains the required hashed username
//     and password for comparison.
//   - givenUser: A string representing the provided username which should be
//     compared with the required username stored in MiddlewareBuilder.
//   - givenPassword: A string representing the provided password which
//     should be compared with the required password stored in MiddlewareBuilder.
//
// Returns:
//   - A boolean value indicating the result of the credentials check. If both
//     the username and password match with the required username and password
//     respectively, it will return true. Otherwise, it will return false.
func checkCredentials(m *MiddlewareBuilder, givenUser, givenPassword string) bool {
	// Compare the hashed given username and required username.
	userMatch := bcrypt.CompareHashAndPassword(m.RequiredUserHash, []byte(givenUser)) == nil
	// Compare the hashed given password and the required password.
	passMatch := bcrypt.CompareHashAndPassword(m.RequiredPasswordHash, []byte(givenPassword)) == nil

	// Return true only if both username and password match.
	return userMatch && passMatch
}

// unauthorized is a helper function that sets and sends an HTTP 401 Unauthorized
// status response. This response indicates that the client must authenticate
// itself to get the requested response.
//
// Parameters:
//   - w: An http.ResponseWriter object that is used to send an HTTP response back
//     to the client making the request.
//   - realm: A string indicating the scope of protection in the event of
//     unauthorized access. This is concatenated to the WWW-Authenticate header.
//
// Returns:
// - Nothing. The function's purpose is to send a response, not to return anything.
func unauthorized(w http.ResponseWriter, realm string) {
	// Set the 'WWW-Authenticate' header of the response. This header defines the
	// auth method that should be used to gain access to a resource.
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)

	// Set the status of the response. StatusUnauthorized (401) implies that
	// the client tried to operate on a protected resource without providing
	// the proper authorization headers in the request.
	w.WriteHeader(http.StatusUnauthorized)

	// Use the Fprint function to write a string to the response writer. The
	// string simply communicates to the client that they are unauthorized.
	// Ignore the error and number of bytes written returned by the Fprint function.
	_, _ = fmt.Fprint(w, "Unauthorized")
}

// parseBasicAuth is a helper function to parse the 'Authorization' header of an incoming HTTP request
// when the 'Basic' scheme is used for HTTP auth. It expects the 'auth' parameter
// to contain the 'Authorization' header from an HTTP request.
//
// Parameters:
// - auth: A string representing the 'Authorization' header from an HTTP request.
//
// Returns:
//   - username: A string that contains the parsed username from the 'Authorization' header.
//   - password: A string that contains the parsed password from the 'Authorization' header.
//   - ok: A boolean flag representing the status of the parsing operation. Returns 'true'
//     if the header was successfully parsed. Otherwise, it will be 'false'.
func parseBasicAuth(auth string) (username, password string, ok bool) {
	// Define the prefix that indicates the use of Basic Authentication.
	const prefix = "Basic "

	// If the auth string does not start with the prefix, return the zero values of the return types.
	if !strings.HasPrefix(auth, prefix) {
		return
	}

	// Decode the base64 encoded auth string excluding the prefix.
	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])

	// If an error occurred during decoding, log the error and return the zero values of the return types.
	if err != nil {
		log.Println("parseBasicAuth: Auth header is not Basic Auth")
		return
	}

	// Split the decoded auth string into a username and password component.
	credentials := bytes.SplitN(decoded, []byte(":"), 2)

	// If the credentials do not contain both a username and password, log the error
	// and return the zero values of the return types.
	if len(credentials) != 2 {
		log.Printf("parseBasicAuth: Failed to decode auth: %v", err)
		return
	}

	// Return the decoded and splitted credentials as well as 'true' to indicate the successful parsing.
	return string(credentials[0]), string(credentials[1]), true
}
