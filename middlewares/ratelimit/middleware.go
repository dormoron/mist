package ratelimit

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/ratelimit"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// MiddlewareBuilder is a struct type that encapsulates the logic for
// building rate limiting middleware which can be used in different contexts,
// such as an HTTP Server to control the rate of incoming requests.
// Fields:
//
//   - limiter: This is an implementation of the Limiter interface, it controls how to check
//     and handle rate limits (e.g., using a sliding window algorithm or auth bucket strategy).
//
//   - keyFn: This is a function that generates a unique key for each request, based on the provided mist.Context.
//     For example, in an HTTP context this may generate keys based on a client IP address or authenticated user ID.
//
//   - logFn: This function is used for logging purposes and can be customized as per your requirements.
//     It receives a string for log level, a string for log message, and a variadic parameter for any additional arguments.
//
//   - retryAfterSec: Integer value specifying the time in seconds a client should wait before retrying
//     when they hit the rate limit.
type MiddlewareBuilder struct {
	limiter       ratelimit.Limiter
	keyFn         func(ctx *mist.Context) string
	logFn         func(level string, msg any, args ...any)
	retryAfterSec int
}

// InitMiddlewareBuilder is a function used to initialize a MiddlewareBuilder instance. It sets up
// the rate limiter, retry timer, default key generation function and the logging function. The function
// can be further customized with MiddlewareOptions.
// Parameters:
//   - limiter: This is the rate limiter instance that controls the rate of incoming requests based on the
//     generated key.
//   - retryAfterSec: This is an integer representing the time to wait before retrying in the event of a rate limit breach.
//   - opts: These are optional arguments that provide further customization to the MiddlewareBuilder. They are
//     functions that accept an all MiddlewareOptions and allow you to set various fields of the MiddlewareBuilder.
//
// Returns:
//   - A pointer to the newly created MiddlewareBuilder.
//
// The InitMiddlewareBuilder function provides initially default implementations for the key generation function
// (keyFn) and logging function (logFn), but these can be replaced with custom implementations using the
// WithKeyGenFunc and WithLogFunc functions.
func InitMiddlewareBuilder(limiter ratelimit.Limiter, retryAfterSec int) *MiddlewareBuilder {
	builder := &MiddlewareBuilder{
		limiter:       limiter,       // Set up the rate limiter with the provided limiter
		retryAfterSec: retryAfterSec, // Set up the retry timer with the provided retryAfterSec
		keyFn: func(ctx *mist.Context) string { // Default key generation function
			var b strings.Builder
			b.WriteString("ip-limiter")
			b.WriteString(":")
			b.WriteString(ctx.ClientIP())
			return b.String()
		},
		logFn: func(level string, msg any, args ...any) { // Default logging function
			v := make([]any, 0, len(args)+2)
			v = append(v, level)
			v = append(v, msg)
			v = append(v, args...)
			log.Println(v...)
		},
	}

	return builder // Return the initialized MiddlewareBuilder
}

// SetKeyGenFunc sets the key generation function to be used by the middleware.
// This function will be called to generate a key for each request, which can be used for various purposes, such as rate limiting or caching.
// Parameters:
// - fn: a function that takes a pointer to a mist.Context and returns a string key.
// Returns:
// - the pointer to the MiddlewareBuilder instance to allow method chaining.
func (b *MiddlewareBuilder) SetKeyGenFunc(fn func(*mist.Context) string) *MiddlewareBuilder {
	b.keyFn = fn // Assign the provided function to the keyFn field.
	return b     // Return the MiddlewareBuilder instance for chaining.
}

// SetLogFunc sets the logging function to be used by the middleware.
// This function will be called whenever the middleware needs to log information, allowing for custom logging implementations.
// Parameters:
// - fn: a function that takes a log level as a string, a message of any type, and optional additional arguments.
// Returns:
// - the pointer to the MiddlewareBuilder instance to allow method chaining.
func (b *MiddlewareBuilder) SetLogFunc(fn func(level string, msg any, args ...any)) *MiddlewareBuilder {
	b.logFn = fn // Assign the provided function to the logFn field.
	return b     // Return the MiddlewareBuilder instance for chaining.
}

// Build is a method of the MiddlewareBuilder type. It returns a middleware that encompasses rate limiting logic.
// The returned middleware is a pipeline unit in the Mist web framework, which is a function accepting the next middleware
// (i.e., next mist.HandleFunc) and returns the resulting middleware function.
// Built middleware does the following on each incoming request:
//  1. It uses the limiter to check if the request rate limit has been exceeded for the current client (using the rate limiter
//     key generated by keyFn from the mist.Context).
//  2. In case the limit function returns an error, it logs an error message, sets the HTTP response status code to 500 (Internal
//     Server Error), and ends the request handling pipeline by not calling the next middleware.
//  3. If the limit is exceeded (as indicated by the 'limited' boolean), it logs a warning message, sets the HTTP response status
//     code to 429 (Too Many Requests), instructs the client when to retry by setting the 'Retry-After' response header, and ends
//     the request-handling pipeline.
//  4. If the rate limit has not been exceeded, it just passes the control to the next middleware in the pipeline by calling
//     next with the mist.Context.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			limited, err := b.limit(ctx) // check if the request rate limit has been exceeded
			if err != nil {              // If there is an error in limiting function, log it and halt request handling by returning an error status code.
				b.logFn("error", "The current limiting detection error: ", err)
				ctx.AbortWithStatus(http.StatusInternalServerError)
				http.Error(ctx.ResponseWriter, "Internal server error", http.StatusInternalServerError)
				return
			}
			if limited { // If the rate limit is exceeded, log a warning, set the response headers accordingly and halt the request handling.
				b.logFn("warn", "The request is blocked")
				ctx.AbortWithStatus(http.StatusTooManyRequests)
				ctx.ResponseWriter.Header().Set("Retry-After", strconv.Itoa(b.retryAfterSec))
				http.Error(ctx.ResponseWriter, "Too many requests, please try again later", ctx.RespStatusCode)
				return
			}
			next(ctx) // Else, pass the control to the next middleware in the pipeline
		}
	}
}

// limit is a method attached to the MiddlewareBuilder type that determines whether a request should be limited
// based on the rate limiter's state and a key derived from the request context.
// Parameters:
//   - ctx: A pointer to the mist.Context, which holds the request and response information within the Mist framework.
//
// Returns:
//   - A boolean indicating whether the request is limited (true if it is, false otherwise).
//   - An error value that will be non-nil if an error occurred during the limit operation.
//
// The method performs the following actions:
//  1. Generates a key using the key function defined in the MiddlewareBuilder.
//  2. If the generated key is an empty string, logs an error message (indicating a potential misconfiguration
//     or problem in the key generation logic) and returns no limitation (false) and no error (nil).
//  3. Otherwise, use the rate limiter to determine if the request associated with the key should be limited,
//     and returns the result.
func (b *MiddlewareBuilder) limit(ctx *mist.Context) (bool, error) {
	key := b.keyFn(ctx) // Generate a key for the request using the key function provided in the MiddlewareBuilder.
	if key == "" {      // Check if the key is an empty string, which indicates a problem in key generation.
		b.logFn("error", "Failed to generate a key") // Log an error message indicating key generation failure.
		return false, nil                            // Return no limitation on the request (false) and no error (nil).
	}
	// Use the rate limiter to decide whether the request with generated key should be limited,
	// b.keyFn(ctx) is called again ensuring the most up-to-date context is used for key generation.
	return b.limiter.Limit(ctx, b.keyFn(ctx))
}
