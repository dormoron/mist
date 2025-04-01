package cookie

import (
	"net/http"
	"time"
)

// PropagatorOptions is a functional option type for configuring instances of Propagator.
// The type defines a function signature that accepts a pointer to Propagator as its sole argument.
// Functional options are a pattern used to simplify and make more readable the configuration of objects.
// Instead of constructors with many parameters, functional options allow for more flexible and
// composable configurations. A client can pass zero or more of these functions to configure a
// Propagator as needed when creating or updating it.
//
// Examples of such functions might include options that set internal timeout values, specify header names
// for propagation, or configure other internal settings of a Propagator object.
//
// Usage:
// To use PropagatorOptions, define functions that match the signature of this type.
// When such a function is called with a Propagator instance, it should modify the Propagator's state
// according to the intended configuration. These functions can then be passed to a constructor function
// or method responsible for setting up a new Propagator, where they will be applied in sequence.
//
// Example of a PropagatorOptions function:
//
//	func WithCustomHeader(name string) PropagatorOptions {
//	    return func(p *Propagator) {
//	        p.headerName = name
//	    }
//	}
type PropagatorOptions func(p *Propagator)

// Propagator is a configuration struct used for managing the settings of an HTTP cookie.
// It provides a structured way to specify the name of a cookie and to define custom behaviors for it
// via a functional option. This can be particularly useful when creating middlewares or handlers
// in web applications where controlled manipulation of cookies is necessary.
//
// Fields:
//
//   - cookieName: This is the identifier for the cookie that the Propagator will manage.
//     It is the key used when setting and retrieving the cookie from an HTTP request or response.
//     The name should be chosen to avoid conflicts with other cookies and should be descriptive
//     of the cookie's purpose.
//
//   - cookieOption: This function allows for customization of the http.Cookie object. It can
//     be used to apply a range of configurations such as setting the path, domain, secure flag,
//     HTTP-only flag, expiration, and any other applicable cookie attributes. By passing a function
//     that takes a pointer to a http.Cookie, the Propagator allows for direct modification of the
//     cookie's properties, enabling more flexible and comprehensive control over cookie behavior.
//
// The Propagator struct thus provides a centralized mechanism to define and manage how a
// cookie is handled in web application scenarios. Its flexibility makes it suitable to fit
// varied requirements and helps in keeping the cookie-related code clean and maintainable.
//
// To use Propagator, the developer can instantiate it with the desired cookie name and a custom
// configuration function. This function will be applied each time the Propagator is called upon
// to manipulate a http.Cookie instance, ensuring that all such cookies adhere to the defined settings.
//
// Example:
// // createSecureCookieOption returns a configuration function that sets the Secure
// // and HttpOnly attributes on a http.Cookie, helping to protect the cookie from
// // certain classes of web vulnerabilities.
//
//	func createSecureCookieOption() func(cookie *http.Cookie) {
//	    return func(cookie *http.Cookie) {
//	        // Secure attribute specifies that the cookie should only be transmitted
//	        // over a secure HTTPS connection from the client. When set to true, the
//	        // cookie will not be sent by the client to the server over an unsecure
//	        // connection (HTTP).
//	        cookie.Secure = true
//
//	        // HttpOnly attribute specifies that the cookie is intended to be accessed
//	        // only by HTTP or HTTPS requests. This helps mitigate the threat of
//	        // client-side script accessing the protected cookie (if the browser supports it).
//	        cookie.HttpOnly = true
//	    }
//	}
//
// // With this setup, a developer can create a new Propagator like so:
//
//	securePropagator := Propagator{
//	    cookieName: "session_token",
//	    cookieOption: createSecureCookieOption(),
//	}
//
// The above example showcases how to define a Propagator that will manage a "session_token" cookie,
// ensuring that it is configured to be Secure and HttpOnly compliant for improved security.
type Propagator struct {
	cookieName   string
	cookieOption func(cookie *http.Cookie)
}

// InitPropagator initializes a new instance of a Propagator object with default settings.
// The function can take a variable number of PropagatorOptions, which are functional options that
// modify the Propagator's configuration. If no options are passed, the Propagator will be
// initialized with a default cookie name 'sessionId' and an empty cookieOption, which by default
// doesn't modify the HTTP cookie.
//
// Parameters:
//   - opts ...PropagatorOptions: A variadic parameter that allows passing zero or more functions
//     that conform to the PropagatorOptions type. Each PropagatorOptions is a function that takes
//     a *Propagator and returns nothing, intended to modify the Propagator's configuration.
//
// Returns:
//   - *Propagator: A pointer to the newly initialized Propagator instance with its configuration
//     set according to the passed options.
func InitPropagator(opts ...PropagatorOptions) *Propagator {
	// Initialize a new Propagator with the default configuration.
	// Default cookieName is set to 'sessionId'.
	// Default cookieOption is an empty function that accepts an *http.Cookie but doesn't modify it.
	res := &Propagator{
		cookieName: "sessionId",
		cookieOption: func(cookie *http.Cookie) {
			// This is an intentionally empty function, which serves as a placeholder,
			// making it possible for the user to specify custom cookie options later.
		},
	}

	// Iterate over each passed option in the 'opts' variadic parameter.
	// Each option is a function that modifies the 'res' (the Propagator instance) in some way.
	for _, opt := range opts {
		opt(res) // Apply the option function to the Propagator instance.
	}

	// Return the pointer to the Propagator instance with its now fully configured state.
	return res
}

// WithCookieName is a functional option helper that creates a PropagatorOptions function,
// which sets the cookieName field of a Propagator struct when applied. This is a common design
// pattern in Go for configuring instances with optional parameters in a flexible and clean manner.
// Instead of having a complex constructor or a multitude of setters for each parameter, you can use
// options functions that apply the desired configurations to the struct.
//
// Parameters:
// - name string: The name of the cookie you wish to set for your Propagator
//
// Returns:
//   - PropagatorOptions: A function that can be passed to a Propagator's configuration.
//     When called, it will set the Propagator's cookieName to the
//     specified 'name' parameter.
//
// Example usage:
// p := &Propagator{}
// nameOption := WithCookieName("user_session")
// nameOption(p)
// This will set the cookieName of the Propagator 'p' to "user_session".
func WithCookieName(name string) PropagatorOptions {
	// Return a function that conforms to the PropagatorOptions type. This returned function
	// takes a pointer to a Propagator as its parameter and sets the Propagator's cookieName field.
	return func(p *Propagator) {
		// Set the cookieName field of the Propagator to the provided 'name' string.
		p.cookieName = name
	}
}

// WithCookieOption returns a PropagatorOptions function that, when applied, configures a Propagator instance
// with a custom cookie option modification function. This allows the caller to specify exactly how http.Cookies
// should be modified by the Propagator when it's working with them.
//
// Parameters:
//   - opt: This is a function that takes a pointer to a http.Cookie and modifies it. The modifications can include
//     setting the Secure flag, HttpOnly flag, adjusting the MaxAge property, and so on. The function encapsulates
//     the intended behavior on the cookie, and this behavior is applied each time the Propagator deals with
//     setting or altering a cookie.
//
// Returns:
//   - A function conforming to the PropagatorOptions type. When this function is applied to a Propagator instance,
//     it sets the instance's `cookieOption` field to the `opt` function provided.
func WithCookieOption(opt func(c *http.Cookie)) PropagatorOptions {
	// Return a PropagatorOptions function. This is a higher-order function that takes another function as an argument,
	// and returns a function as a result.
	return func(propagator *Propagator) {
		// Within the returned PropagatorOptions function, assign the provided opt function to the propagator's
		// cookieOption field. This operation modifies the behavior of the Propagator specifically with how it
		// will handle http.Cookies.
		propagator.cookieOption = opt
	}
}

// Inject attaches a cookie with a specified ID to the HTTP response writer provided.
// This method is a part of the Propagator struct type and is used for setting a
// session cookie into the HTTP response that will be sent back to the client.
//
// Parameters:
// - id string: The session identifier value that will be stored in the cookie.
// - writer http.ResponseWriter: The HTTP response writer to which the session cookie will be added.
//
// Returns:
//   - error: This function always returns nil as per the current implementation, implying
//     that no error has been encountered during the injection process. If future
//     implementations include error checks (e.g., response writer validation),
//     this may return an actual error.
//
// Example usage:
// propagator := InitPropagator() // create a new propagator
// err := propagator.Inject("session_id_value", responseWriter)
//
//	if err != nil {
//	   // handle error
//	}
func (p *Propagator) Inject(id string, writer http.ResponseWriter) error {
	// Create a new HTTP cookie with the name from the Propagator's cookieName field
	// and the value provided in the 'id' parameter.
	cookie := &http.Cookie{
		Name:  p.cookieName, // Set the Name field of the cookie to the Propagator's cookieName.
		Value: id,           // Set the Value field of the cookie to the 'id' parameter.
	}

	// Apply the cookie configuration defined in the Propagator's cookieOption function.
	// This is where any additional settings or overrides should be made to the cookie before it is set.
	p.cookieOption(cookie)

	// Set the cookie on the HTTP response writer object.
	// This will append a 'Set-Cookie' header to the response that will be sent to the client.
	http.SetCookie(writer, cookie)

	// Return nil, indicating no error has occurred during the cookie injection process.
	// If needed in the future, error handling can be implemented here if something goes wrong
	// with setting the cookie.
	return nil
}

// Extract retrieves the value of the cookie identified by the Propagator's cookieName
// from the provided HTTP request. If the cookie is present, its value is returned;
// otherwise, an error is returned indicating the cookie could not be found.
//
// Parameters:
// - req *http.Request: The HTTP request from which the cookie will be extracted.
//
// Returns:
//   - string: The value of the extracted cookie. This will be the session identifier if
//     the cookie is found.
//   - error: An error message if the cookie is not found in the request. If the cookie
//     is found, nil is returned.
//
// Example usage:
// propagator := InitPropagator() // create a new propagator
// sessionID, err := propagator.Extract(request)
//
//	if err != nil {
//	    // handle error, such as a missing cookie
//	} else {
//
//	    // use the sessionID as needed
//	}
func (p *Propagator) Extract(req *http.Request) (string, error) {
	// Attempt to retrieve the cookie by the name stored in the Propagator's cookieName field from the request.
	cookie, err := req.Cookie(p.cookieName)
	// If there's an error (e.g., the cookie does not exist), return an empty string and the error.
	if err != nil {
		return "", err
	}
	// If the cookie is found with no errors, return its value and nil for the error.
	return cookie.Value, nil
}

// Remove creates a cookie with the same name as that stored in the Propagator's cookieName field
// but sets its MaxAge to -1, effectively instructing the client's browser to delete the cookie.
// This method is used to remove a client's session cookie by adding this 'expired' cookie to
// the HTTP response.
//
// Parameters:
//   - writer http.ResponseWriter: The HTTP response writer to which the 'expired' cookie
//     will be added in order to instruct the client to remove the cookie.
//
// Returns:
//   - error: This function always returns nil as per the current implementation, signifying
//     the operation was successful without any errors. If error handling is added in
//     the future, this could potentially return actual errors if they occur.
//
// Example usage:
// propagator := InitPropagator() // Initializes a new Propagator
// err := propagator.Remove(responseWriter)
//
//	if err != nil {
//	   // handle error
//	}
func (p *Propagator) Remove(writer http.ResponseWriter) error {
	// Create an HTTP cookie with a negative MaxAge to ensure the browser deletes it.
	cookie := &http.Cookie{
		// Set the Name field of the cookie to the Propagator's cookieName.
		Name: p.cookieName,
		// Setting MaxAge to -1 causes the cookie to be deleted immediately by the client.
		MaxAge: -1,
	}

	// Apply the cookie configuration defined in the Propagator's cookieOption function.
	// This is where any additional settings or overrides should be made to the cookie before it is set.
	p.cookieOption(cookie)

	// Set the 'expired' cookie on the HTTP response writer object.
	// This adds a 'Set-Cookie' header to the response with the 'expired' cookie.
	http.SetCookie(writer, cookie)

	// Return nil, indicating no error has occurred during the cookie removal process.
	// In the future, error handling could be included here if, for instance, the response
	// writer is not in a valid state to set headers.
	return nil
}

// CookiePropagator 基于Cookie的会话ID传播器
type CookiePropagator struct {
	name     string
	path     string
	domain   string
	maxAge   int
	secure   bool
	httpOnly bool
	sameSite http.SameSite
}

// NewPropagator 创建新的Cookie传播器
func NewPropagator(name string, opts ...PropagatorOption) *CookiePropagator {
	p := &CookiePropagator{
		name:     name,
		path:     "/",
		maxAge:   3600, // 1小时
		secure:   true,
		httpOnly: true,
		sameSite: http.SameSiteStrictMode,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// PropagatorOption Cookie传播器配置函数
type PropagatorOption func(*CookiePropagator)

// WithPath 设置Cookie路径
func WithPath(path string) PropagatorOption {
	return func(p *CookiePropagator) {
		p.path = path
	}
}

// WithDomain 设置Cookie域名
func WithDomain(domain string) PropagatorOption {
	return func(p *CookiePropagator) {
		p.domain = domain
	}
}

// WithMaxAge 设置Cookie最大有效期（秒）
func WithMaxAge(maxAge int) PropagatorOption {
	return func(p *CookiePropagator) {
		p.maxAge = maxAge
	}
}

// WithSecure 设置Cookie是否只通过HTTPS传输
func WithSecure(secure bool) PropagatorOption {
	return func(p *CookiePropagator) {
		p.secure = secure
	}
}

// WithHTTPOnly 设置Cookie是否禁止JavaScript访问
func WithHTTPOnly(httpOnly bool) PropagatorOption {
	return func(p *CookiePropagator) {
		p.httpOnly = httpOnly
	}
}

// WithSameSite 设置Cookie的SameSite属性
func WithSameSite(sameSite http.SameSite) PropagatorOption {
	return func(p *CookiePropagator) {
		p.sameSite = sameSite
	}
}

// Inject 将会话ID注入到响应中的Cookie
func (p *CookiePropagator) Inject(id string, writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:     p.name,
		Value:    id,
		Path:     p.path,
		Domain:   p.domain,
		MaxAge:   p.maxAge,
		Secure:   p.secure,
		HttpOnly: p.httpOnly,
		SameSite: p.sameSite,
	}
	http.SetCookie(writer, cookie)
	return nil
}

// Extract 从请求中的Cookie提取会话ID
func (p *CookiePropagator) Extract(req *http.Request) (string, error) {
	cookie, err := req.Cookie(p.name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// Remove 从响应中移除会话Cookie
func (p *CookiePropagator) Remove(writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:     p.name,
		Value:    "",
		Path:     p.path,
		Domain:   p.domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   p.secure,
		HttpOnly: p.httpOnly,
		SameSite: p.sameSite,
	}
	http.SetCookie(writer, cookie)
	return nil
}
