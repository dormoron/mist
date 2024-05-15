package redislimit

import (
	"fmt"
	"github.com/dormoron/mist"
	"github.com/redis/go-redis/v9"
	"go.uber.org/atomic"
	"net/http"
)

// MiddlewareBuilder is a struct that encapsulates essential configurations
// for constructing middleware. This configuration aids in managing the request
// flow, resource access, and logging within an application, especially concerning
// rate-limiting and interaction with Redis for command execution.
// Fields:
//   - maxActive: A pointer to an atomic Int64 that specifies the maximum number
//     of active processing requests allowed at any given time. Utilizing atomic operation
//     ensures that reading and updating the maxActive value is thread-safe, crucial for
//     high-concurrency scenarios, thereby preventing overloading the system.
//   - key: A string value that serves as an identifier for the middleware instance.
//     The key can be used to uniquely identify and differentiate between multiple middleware configurations,
//     particularly when interfacing with external systems like Redis for storing and retrieving state information.
//   - cmd: An interface of type redis.Cmdable provided by the go-redis package. This interface allows the
//     MiddlewareBuilder to execute Redis commands. It's a flexible way to interact with Redis, enabling
//     operations such as setting rate limits or counting requests, thus integrating seamlessly with Redis
//     to manage application data or state.
//   - logFn: A function signature that defines a custom logging function. This function is designed to accept
//     a message of any type along with an optional variadic list of additional arguments. This flexibility allows
//     developers to log a wide range of information, aiding in monitoring and troubleshooting by providing insights
//     into the middleware's operations. The function can be customized to log to various outputs (e.g., console,
//     file, external monitoring service) and can be tailored to include specific information relevant to the
//     application's needs.
//
// The MiddlewareBuilder struct plays a crucial role in setting up middleware that can manage request rates,
// perform logging, and execute commands in Redis, providing a solid foundation for middleware configuration
// and execution in web applications.
type MiddlewareBuilder struct {
	maxActive *atomic.Int64

	key string

	cmd redis.Cmdable

	logFn func(msg any, args ...any)
}

// InitMiddlewareBuilder is a function designed to initialize and return
// an instance of MiddlewareBuilder with specific configurations. This function
// serves as a constructor-like mechanism, simplifying the instantiation process
// by bundling the necessary parameters into a single call. It allows the configuration
// of essential functionalities for managing request rate limiting and logging
// through Redis commands.
// Parameters:
//
//   - cmd: Accepts an interface of type redis.Cmdable which represents the Redis commands
//     that the middleware will be able to execute. This parameter enables the MiddlewareBuilder
//     to interact with Redis, a fast, open-source, in-memory key-value data store,
//     for operations such as setting rate limits or counting requests.
//
//   - maxActive: A 64-bit integer (int64) defining the maximum number of active requests
//     that the middleware allows to be processed concurrently. This limit is crucial
//     for preventing system overload and ensuring that the application can handle
//     high concurrency without degrading performance.
//
//   - key: A string that serves as a unique identifier for the middleware instance.
//     This key helps to uniquely identify the middleware configuration,
//     especially useful when multiple instances or types of middleware are
//     used within an application.
//
// Returns:
//   - An instance of MiddlewareBuilder configured with the provided parameters.
//     This instance encapsulates the functionality for rate limiting and logging through
//     interactions with Redis. It stands ready to be utilized in managing the flow
//     and processing of requests within an application's middleware stack.
//
// Usage:
// The function is typically used to instantiate a MiddlewareBuilder with carefully
// chosen parameters that match the application's requirements for rate limiting
// and logging. Once created, the MiddlewareBuilder instance can be integrated into
// the application's middleware flow, contributing to efficient request handling
// and enhanced monitoring capabilities.
func InitMiddlewareBuilder(cmd redis.Cmdable, maxActive int64, key string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		maxActive: atomic.NewInt64(maxActive), // Initializes an atomic Int64 with the value of maxActive,
		// ensuring thread-safe manipulation of this limit.

		key: key, // Sets the unique identifier key for this middleware instance.

		cmd: cmd, // Establishes the Redis command interface for executing operations in Redis.

		logFn: func(msg any, args ...any) { // Defines a default logging function that can be overridden.
			fmt.Printf("%v info message: %v \n", msg, args) // The default function prints formatted log messages
			// to standard output. It accepts a message and variadic
			// arguments to provide flexible logging.
		},
	}
}

// SetLogFunc is a method on the MiddlewareBuilder struct that allows for the
// configuration of a custom logging function. This method is designed to be used
// for setting the internal logging function (`logFn`) of a MiddlewareBuilder instance,
// which is responsible for outputting log messages.
// Parameters:
//   - fun: A function with a signature that accepts a `msg` of any type and a
//     variadic series of arguments `args`. The `msg` parameter is intended
//     to be the primary log message, and `args` allows for additional context
//     or data to be provided alongside the log message. This function parameter
//     can be any user-defined function that meets the provided signature, enabling the
//     capture and handling of log messages according to user requirements.
//
// Returns:
//   - A pointer to the MiddlewareBuilder instance (*MiddlewareBuilder). This
//     enables method chaining, a common design pattern in Go which allows for
//     multiple method calls to be linked together in a fluent and readable manner.
//     In this case, after setting the logging function, the caller can continue
//     to call other configuration methods on the MiddlewareBuilder instance without
//     needing to start a new statement.
//
// Usage:
// This method is typically invoked to override the default logging behavior of a
// MiddlewareBuilder instance. By passing in a custom logging function, the user
// can define how log messages will be handled, such as writing them to a file, sending
// them to a log aggregation service, or formatting them in a specific way. The flexibility
// provided by this method allows for robust and versatile logging configurations within
// middleware implementations.
func (b *MiddlewareBuilder) SetLogFunc(fun func(msg any, args ...any)) *MiddlewareBuilder {
	b.logFn = fun // Assigns the `fun` parameter to the instance's `logFn` field.
	return b      // Returns the instance itself, enabling method chaining.
}

// Build is a method defined on the MiddlewareBuilder struct that compiles
// and returns a new middleware function. This function is designed to integrate seamlessly
// with the 'mist' web framework, providing a mechanism to enforce rate limiting on HTTP requests
// by utilizing Redis to track the current count of active requests.
// Returns:
//   - A middleware function of type mist.Middleware. This function takes another function
//     of type mist.HandleFunc as its parameter (referred to as `next`), and returns a function
//     of the same type. The returned function is where the core logic of the middleware resides,
//     capturing the essence of a middleware by taking an HTTP request (encapsulated in
//     `ctx *mist.Context`), processing it, and deciding whether to pass the request along the
//     middleware chain or to terminate it.
//
// The Build method's return is a closure that encapsulates:
//   - Rate limiting logic: Before invoking the `next` handler, it checks against a Redis counter
//     to ensure the number of active requests does not exceed a predefined maximum (`b.maxActive`).
//     This count is incremented at the start and decremented at the end of each request, ensuring an
//     up-to-date tally of concurrent requests.
//   - Error handling: Properly handles scenarios where Redis commands (increment/decrement) fail,
//     logging errors and aborting the request with appropriate HTTP status codes without
//     proceeding further in the handler chain.
//   - Next handler invocation: If the rate limit check passes, the middleware invokes the `next`
//     function, effectively passing control to the next middleware in the chain or the final
//     request handler if there are no further middlewares.
//
// Usage:
// This middleware is typically used in web applications that require rate limiting to protect resources
// against overload by too many concurrent requests. It's particularly useful in scenarios where
// application performance and availability are of concern, helping to ensure a fair use policy
// and prevent system abuse.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// Increment the Redis counter for current active requests.
			currentCount, err := b.cmd.Incr(ctx.Request.Context(), b.key).Result()
			if err != nil {
				// Log and abort the request if there is an error incrementing the Redis counter.
				b.logFn("error incrementing redis counter", err)
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			// Ensure the Redis counter is decremented after handling the request.
			defer func() {
				if err = b.cmd.Decr(ctx.Request.Context(), b.key).Err(); err != nil {
					// Log if there is an error decrementing the Redis counter.
					b.logFn("error decrementing redis counter", err)
				}
			}()

			// Check if the current count exceeds the maximum allowed active requests.
			if currentCount > b.maxActive.Load() {
				// Log and abort the request if rate limiting is in effect.
				b.logFn("rate limiting in effect")
				ctx.AbortWithStatus(http.StatusTooManyRequests)
				return
			}

			// Proceed to the next middleware or final handler if rate limit check passes.
			next(ctx)
		}
	}
}
