package mist

// Middleware represents a function type in Go that defines the structure of a middleware function.
// In the context of web servers or other request-handling applications, middleware is used to process
// requests before reaching the final request handler, allowing for pre-processing like auth,
// logging, or any other operation that should be performed before or after the main processing of a request.
//
// The type is defined as a function that takes one HandleFunc as its parameter (often referred to as 'next')
// and returns another HandleFunc. The HandleFunc inside the parentheses is the next function in the chain
// that the middleware will call, while the HandleFunc being returned is the modified or "wrapped" version of that function.
//
// A typical middleware will perform some actions, then call 'next' to pass control to the subsequent middleware or the final
// handler, potentially perform some actions after 'next' has returned, and finally return the result of 'next'. By doing so,
// it forms a chain of middleware functions through which the request flows.
//
// The Middleware type is designed to be flexible and composable, making the construction of an ordered sequence
// of middleware functions straightforward and modular.
//
// Parameters:
//   - 'next': The HandleFunc to wrap with additional behavior. This is the function that would normally handle
//     the request or be the next middleware in line.
//
// Return Value:
// - A HandleFunc that represents the result of adding the middleware's behavior to the 'next' function.
//
// Usage:
//   - Middleware functions are typically used with a router or a server to handle HTTP requests.
//   - They are chained together so that a request goes through a series of middleware before finally being
//     handled by the main processing function.
//
// Considerations:
//   - When designing middleware, one should ensure that no necessary 'next' handlers are skipped inadvertently.
//     Unless it's intentional (e.g., an authorization middleware stopping unauthorized requests), a middleware
//     should usually call 'next'.
//   - Be careful with error handling in middleware. Decide whether to handle and log errors within the middleware
//     itself or pass them along to be handled by other mechanism.
//   - Middleware functions should avoid altering the request unless it is part of its clear responsibility,
//     such as setting context values or modifying headers that pertain to middleware-specific functionality.
type Middleware func(next HandleFunc) HandleFunc
