package mist

import (
	"encoding/json"
	"github.com/dormoron/mist/internal/errs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Context is a custom type designed to carry the state and data needed to process
// an HTTP request within the application. It provides context around the HTTP
// request, including both the request and response information, as well as additional
// data needed by handlers to fulfill the request.
type Context struct {
	// Request is the original http.Request object. It represents the HTTP request
	// received by the server and contains properties such as the URL, headers,
	// query parameters, etc. Handlers can access this to read request data.
	Request *http.Request

	// ResponseWriter is an interface used to respond to an HTTP request.
	// It is the writer that sends the HTTP response back to the client. Handlers use
	// this to write the response headers and response body.
	ResponseWriter http.ResponseWriter

	// PathParams contains any URL path parameters matched by the routing mechanism.
	// For example, in a route defined as "/users/:id", the "id" would be available
	// in this map for the path "/users/123".
	PathParams map[string]string

	// Keys is a map for storing arbitrary data that can be shared across different
	// parts of the application during the request lifecycle. It is useful for setting
	// and getting values that are pertinent to the current HTTP request.
	Keys map[string]any

	// mutex is a read-write mutex to synchronize access to the Keys map, ensuring
	// thread-safety in concurrent environments.
	mutex sync.RWMutex

	// queryValues are all the URL query parameter values extracted from the
	// request URL. This uses the standard `url.Values` type which is essentially
	// a map with string keys and a slice of strings as the value, since a single
	// key can have multiple values.
	queryValues url.Values

	// MatchedRoute is the pattern of the route that matched the current request.
	// For example, if the request is to "/users/view" and a "/users/:action" route
	// matches it, this field will hold that pattern "/users/:action".
	MatchedRoute string

	// RespData is a buffer to hold the data that will be written to the HTTP response.
	// This is used to accumulate the response body prior to writing to the
	// ResponseWriter.
	RespData []byte

	// RespStatusCode is the HTTP status code that should be sent with the response.
	// This is used to store the intended status code which will be written to the
	// response during the final handling of the request.
	RespStatusCode int

	// templateEngine is the engine or library used for rendering HTML templates.
	// If the response requires rendering a template, this field holds the instance
	// or interface to the template engine that's used to do that rendering.
	templateEngine TemplateEngine

	// UserValues is a flexible storage area provided for the developer to store
	// any additional values that might be needed throughout the life of the request.
	// It is essentially a map that can hold values of any type, indexed by string keys.
	UserValues map[string]any

	// headerWritten is a flag indicating whether or not the HTTP headers have already
	// been written to the response. This is used to ensure that headers and status code
	// are not written more than once.
	headerWritten bool

	// Aborted is a flag indicating whether the request handling should be stopped.
	// If true, handlers should terminate further processing immediately.
	Aborted bool
}

// Deadline returns the time when the context will be canceled, if any.
// It delegates the call to the underlying standard context associated with the request.
//
// Returns:
//
//	deadline (time.Time): The time when work done on behalf of this context should be canceled.
//	ok (bool): True if a deadline is set, false if no deadline is set.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Request.Context().Deadline()
}

// Done returns a channel that is closed when the context is canceled or times out.
// It delegates the call to the underlying standard context associated with the request.
//
// Returns:
//
//	(<-chan struct{}): A channel that is closed when the context is canceled or times out.
func (c *Context) Done() <-chan struct{} {
	return c.Request.Context().Done()
}

// Err returns an error indicating why the context was canceled, if applicable.
// It delegates the call to the underlying standard context associated with the request.
//
// Returns:
//
//	(error): A non-nil error value after the Done channel is closed.
//	         If the Done channel is not yet closed, returns nil.
func (c *Context) Err() error {
	return c.Request.Context().Err()
}

// Value retrieves the value associated with the provided key.
// If the key is a string, it first attempts to fetch the value from the custom
// context-specific storage. If the key is not found or is not a string,
// it falls back to the standard context's Value method.
//
// Parameters:
//
//	key (any): The key to look up in the context. If this is a string,
//	           it attempts to fetch the value from the custom context-specific storage.
//
// Returns:
//
//	(any): The value associated with the key, if found; otherwise nil.
func (c *Context) Value(key any) any {
	if keyAsString, ok := key.(string); ok {
		if val, exists := c.Get(keyAsString); exists {
			return val
		}
	}
	return c.Request.Context().Value(key)
}

// writeHeader sends an HTTP response header with the provided status code
// if the header has not been written yet. It ensures that the WriteHeader
// method of the ResponseWriter is called only once during the lifecycle
// of a single HTTP request.
//
// This method belongs to the Context structure and it handles the state of
// the response header using an internal flag (headerWritten).
//
// Params:
//
//	statusCode int - The HTTP status code to be sent with the response header.
func (c *Context) writeHeader(statusCode int) {
	if c.Aborted {
		return
	}

	// Check if the header has already been written.
	// The headerWritten field is a boolean and it indicates whether the
	// HTTP status code and headers have already been sent to the client.
	// This check is crucial to avoid writing the header more than once.
	if !c.headerWritten {
		// Use the ResponseWriter from the Context to write the HTTP status code.
		// ResponseWriter.WriteHeader sends the HTTP response header with the
		// status code provided. If called more than once, it would typically
		// result in a runtime error, as the HTTP protocol allows for only one
		// set of headers per response.
		c.ResponseWriter.WriteHeader(statusCode)
		c.RespStatusCode = statusCode

		// Set the headerWritten flag to true to indicate that the header
		// has now been written for this request.
		// This ensures that subsequent calls to writeHeader within the same
		// request will not attempt to write the header again.
		c.headerWritten = true
	}
}
func (c *Context) AbortWithStatus(code int) {
	if c.Aborted {
		return
	}
	c.writeHeader(code)
	c.Aborted = true
}

// Render processes a template and populates it with dynamic data provided by the 'data' parameter.
// The method first attempts to render the specified template by calling the rendering engine bound to the context.
//
// Parameters:
//   - 'templateName' is the name of the template that should be rendered. It is expected that this template has been
//     defined and is recognizable by the template engine.
//   - 'data' is an interface{} type, which means it can accept any value that conforms to Go's empty interface. This is
//     the dynamic content that will be injected into the template during rendering.
//
// The rendered template output is captured and assigned to 'c.RespData'. This output will typically be HTML or another
// text format suitable for the client's response.
//
// If the rendering operation is successful, the response status code is set to HTTP 200 (OK) indicating the request has
// been successfully processed and the client can expect valid content.
//
// If an error occurs during the rendering operation (such as if the template is not found, or if there is a problem
// with the template's syntax), the response status code is set to HTTP 500 (Internal Server Error), and the error
// is returned to the caller. This error should be handled appropriately by the caller, possibly by logging it or
// presenting a user-friendly error message to the end-user.
//
// Return Value:
// Returns 'nil' if rendering succeeds without any error, otherwise an error object describing the rendering failure is returned.
func (c *Context) Render(templateName string, data any) error {
	var err error
	// Use the template engine to render the template with the provided data.
	c.RespData, err = c.templateEngine.Render(c.Request.Context(), templateName, data)
	if err != nil {
		// On error, set the HTTP status to 500 and return the error.
		c.RespStatusCode = http.StatusInternalServerError
		return err
	}
	// On success, set the HTTP status to 200.
	c.RespStatusCode = http.StatusOK
	return nil
}

// SetCookie adds a Set-Cookie header to the response of the current HTTP context. This method is used to send cookies from the server to the client's web browser.
//
// Parameters:
// - 'ck' is a pointer to an 'http.Cookie' object which represents the cookie you want to set. This object includes various fields that define the properties of a cookie, such as Name, Value, Path, Domain, Expires, and so on.
//
// This function does not return any value or error, as it directly manipulates the HTTP headers of the response.
// It's essential to call SetCookie before writing any response body to the client because HTTP headers cannot be modified after the body starts to be written.
//
// 'http.SetCookie' is a standard library function that ensures the correct formatting of the Set-Cookie header.
// When calling this method, 'c.ResponseWriter' is used to gain access to the response writer associated with the current HTTP request handled by the context 'c'.
//
// Usage:
//   - This method is typically called within an HTTP handler function where you have an instance of the Context 'c' available.
//     It is part of managing session data, sending tracking cookies, or setting any other kind of cookie data required by the application.
//
// Example:
// - To set a session cookie with a session identifier:
//
//	sessionCookie := &http.Cookie{
//	   Name:    "session_id",
//	   Value:   "abc123",
//	   Expires: time.Now().Add(24 * time.Hour),
//	   Path:    "/",
//	}
//	c.SetCookie(sessionCookie)
//
// Note:
// - If you need to set multiple cookies, you would call this method multiple times, passing in each cookie as a separate 'http.Cookie' object.
func (c *Context) SetCookie(ck *http.Cookie) {
	http.SetCookie(c.ResponseWriter, ck)
}

// RemoteIP extracts the remote IP address from the context's request.
// It uses the RemoteAddr field from the request, which typically contains both the IP address and port.
// This method extracts and returns just the IP address part.
// Returns:
// - string: The remote IP address, or an empty string if extraction fails.
func (c *Context) RemoteIP() string {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	// Split the remote address into host and port.
	if err != nil {
		return "" // Return an empty string if there's an error during the split.
	}
	return ip // Return the extracted IP address.
}

// ClientIP attempts to determine the client's IP address from the request headers in the following order:
// 1. X-Forwarded-For header (a common header used in proxy setups).
// 2. X-Real-IP header (another header used in some proxy setups).
// 3. If neither of the above headers are present, it falls back to the remote IP address derived from the RemoteAddr field.
// Returns:
// - string: The determined client IP address.
func (c *Context) ClientIP() string {
	// Check the X-Forwarded-For header first (contains a comma-separated list of IP addresses).
	xForwardedFor := c.Request.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Extract the first IP address from the list and return it.
		ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
		if ip != "" {
			return ip // Return the first IP address from the X-Forwarded-For header if present and non-empty.
		}
	}

	// Check the X-Real-IP header next.
	xRealIP := c.Request.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP) // Return the IP address from the X-Real-IP header if present.
	}

	// If neither header is present, fall back to the remote IP address.
	return c.RemoteIP()
}

// RespondSuccess sends a JSON response with an HTTP status code 200 (OK) to the client.
// This is a convenience method that wraps around the more general 'RespJSON' method, abstracting the common case of responding with an OK status.
//
// Parameters:
//   - 'val' is the data of any type that will be serialized into JSON format. The 'val' can be any Go data structure, including structs, maps, slices, and primitive types.
//     It is important that the 'val' can be marshaled into JSON; otherwise, the serialization will fail.
//
// Internally, 'RespJSONOK' calls the 'RespJSON' method on the same context instance 'c', passing in 'http.StatusOK' as the status code and 'val' as the value
// to be serialized.
//
// The 'RespJSON' method handles the serialization of 'val' to JSON, sets the "Content-Type" response header to "application/json",
// and writes the JSON-encoded data to the response along with the provided HTTP status code.
//
// Return Value:
// - If the serialization into JSON is successful and the response is written to the client, it returns 'nil' indicating no error occurred.
// - In case of an error during JSON marshaling or writing to the response, it returns an error detailing the issue encountered.
//
// Usage:
//   - This method is typically used within an HTTP handler function when the server needs to send a JSON response back to the client with a 200 OK status code.
//     This is usually the case when a request has been processed successfully, and server needs to inform the client of the success, often with accompanying data.
//
// Example:
// - To send a simple success message in JSON format:
//
//	type response struct {
//	   Message string `json:"message"`
//	}
//
//	resp := response{Message: "Data processed successfully"}
//	err := c.RespJSONOK(resp)
//	if err != nil {
//	   // handle error
//	}
//
// Note:
// - This method should be called after all response headers and status codes are set, and before any calls to write body content directly, as it will write data to the body and set headers.
func (c *Context) RespondSuccess(val any) error {
	return c.RespondWithJSON(http.StatusOK, val)
}

// RespondWithJSON sends a JSON-formatted response to the client with a specified HTTP status code.
// It converts the provided value to JSON, sets the response headers, and writes the JSON data to the response.
//
// Parameters:
//   - 'status' is an integer that represents the HTTP status code to send with the response. Standard HTTP status codes should be used
//     (e.g., 200 for OK, 400 for Bad Request, 404 for Not Found, etc.).
//   - 'val' is an interface{} (any type) that holds the data to be serialized into JSON. This can be any Go data structure such as structs, slices, maps, etc.
//     The value provided must be a valid input for the json.Marshal function, which means it should be able to be encoded into JSON. Non-exported struct fields will be omitted by the marshaller.
//
// This function performs several actions:
//  1. It uses the 'json.Marshal' function to serialize the 'val' parameter into a JSON-formatted byte slice 'data'. If marshaling fails,
//     it returns the resultant error without writing anything to the response.
//  2. Assuming marshaling is successful, it sets the "Content-Type" header of the response to "application/json" to inform
//     the client that the server is returning JSON-formatted data.
//  3. It sets the "Content-Length" header to the length of the serialized JSON data, which helps the client understand how much data
//     is being transmitted.
//  4. It writes the HTTP status code to the response using WriteHeader(status). This must be done before writing the response body.
//  5. Lastly, it assigns the JSON data to 'c.RespData' and the status code to 'c.RespStatusCode' for later use or inspection.
//
// Return Value:
// - If the JSON serialization and writing to the response are successful, it returns 'nil', indicating that the operation completed without error.
// - If an error occurs during JSON serialization, the error is returned, and no further action is taken.
//
// Usage:
// - This method is designed to be used in HTTP handler functions where a JSON response is needed. It abstracts away the common tasks of JSON serialization, header setting, and response writing.
//
// Example:
// - To send an object with an OK (200) status code:
//
//	data := map[string]string{"status": "success"}
//	err := c.RespJSON(http.StatusOK, data)
//	if err != nil {
//	   log.Printf("Error sending JSON response: %v", err)
//	   // handle the error, like sending a 500 Internal Server Error status code
//	}
//
// Note:
//   - It is important to note that once the 'WriteHeader' method is called, it's not possible to change the response status code
//     or write any new headers. Also, care must be taken to ensure that 'RespJSON' is not called after the response body has started to be written
//     by other means, as this would result in an HTTP protocol error.
func (c *Context) RespondWithJSON(status int, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	c.writeHeader(status)
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(data)))
	c.RespData = data
	c.RespStatusCode = status
	return err
}

// BindJSON deserializes the JSON-encoded request body into the provided value.
// It is often used in the context of HTTP handlers to parse incoming JSON data into a Go data structure.
//
// Parameters:
//   - 'val' is a pointer to any Go data structure into which the JSON body of the request will be decoded.
//     This argument must be a non-nil pointer that points to an allocatable or allocated value so that the JSON package
//     can populate the fields of the structure. It must not be nil, as passing nil does not provide a storage location
//     for the decoded data.
//
// The function execution involves the following steps:
//  1. It checks if 'val' is nil. If it is, no storage is allocated for the JSON data to be decoded into, so it returns
//     an error using 'errs.ErrInputNil()', indicating that the input value cannot be nil.
//  2. It checks if 'c.Request.Body' is nil. If the body is nil, there is no data to decode, and it returns an error
//     using 'errs.ErrBodyNil()', signaling that there is no request body to parse.
//  3. It creates a new JSON decoder for the request body and performs the decoding operation using 'decoder.Decode(val)'.
//     If the JSON data in the request body is not well-formed or does not match the expected structure of 'val', json.Decoder
//     will return a relevant error to inform the call site about the issue.
//
// Return Value:
// - If the decoding is successful, it returns 'nil', indicating that the JSON data has been successfully bound to 'val'.
// - If an error occurs during decoding, it returns the error, indicating an issue with input validation or the decoding process.
//
// Usage:
//   - This method is useful when you want to automatically handle the parsing of JSON data into a Go data structure from an HTTP request.
//     It abstracts away the low-level streaming and decoding of JSON.
//
// Example:
// - To bind the JSON body of an HTTP request to a struct:
//
//	type UserInput struct {
//	   Name  string `json:"name"`
//	   Email string `json:"email"`
//	}
//
//	var userInput
//	if err := c.BindJSON(&userInput); err != nil {
//	   // handle the error (e.g., send an HTTP 400 Bad Request response)
//	}
//
// Note:
//   - The 'BindJSON' method should be called before accessing the request body by any other means, as the body is an io.ReadCloser
//     and can generally only be read once. Reading the body elsewhere before calling 'BindJSON' will likely result in an EOF error.
func (c *Context) BindJSON(val any) error {
	if val == nil {
		return errs.ErrInputNil()
	}
	if c.Request.Body == nil {
		return errs.ErrBodyNil()
	}
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(val)
}

// BindJSONOpt deserializes the JSON-encoded request body into the provided value with options to use JSON numbers and disallow unknown fields.
// This function extends the BindJSON method by providing additional decoding options that can be enabled as needed.
//
// Parameters:
//   - 'val' is a pointer to any Go data structure into which the JSON body of the request will be decoded. It must be a non-nil pointer, so it can be populated with decoded data.
//     Passing a nil pointer will result in an error since there would be no allocated structure to decode the data into.
//   - 'useNumber' is a boolean option that, when set to true, tells the decoder to decode numbers into an interface{} as a json.Number instead of as a float64.
//     This is useful when precision for large integers is necessary or when the numeric values need to be unmarshalled into a specific typed variable later.
//   - 'disableUnknown' is a boolean option that, when set to true, causes the decoder to return an error if it encounters any unknown fields in the JSON data
//     that do not match any fields in the target Go data structure. This can be useful for strict input validation.
//
// The function performs the following steps:
//  1. It verifies that 'val' is not nil. If it is nil, it returns an error signaling that a valid pointer is required.
//  2. It checks if 'c.Request.Body' is nil, which would indicate that there is no data to decode, and returns an error if this is the case.
//  3. It initializes a JSON decoder for the request body. The decoder then checks the 'useNumber' and 'disableUnknown' options to configure its behavior accordingly:
//     a. If 'useNumber' is true, it configures the decoder to treat numbers as json.Number types instead of defaulting them to float64.
//     b. If 'disableUnknown' is true, it configures the decoder to reject unknown fields.
//  4. It attempts to decode the JSON request body into 'val'. If the decoding is unsuccessful (for example, due to JSON syntax errors,
//     mismatch between the JSON data and 'val' structure, etc.), it returns the error resulting from the decoder.
//
// Return Value:
// - Returns 'nil' if the JSON request body has been successfully decoded into 'val'.
// - Returns an error if 'val' is nil, if there is no request body, or if the JSON decoding process encounters an error.
//
// Usage:
// - This method is useful for controlling the JSON decoding process in HTTP handlers where there may be a need for more strict or lenient JSON parsing.
//
// Example:
// - To bind a JSON body to a struct with strict type preservation and unknown field rejection:
//
//	type UserInput struct {
//	   Name string `json:"name"`
//	   Age  json.Number `json:"age"`
//	}
//
//	var userInput
//	err := c.BindJSONOpt(&userInput, true, true)
//	if err != nil {
//	   // handle the error (e.g., send an HTTP 400 Bad Request response)
//	}
//
// Note:
//   - Similar to BindJSON, BindJSONOpt should be called before any other form of accessing the request body is performed, as it is an io.ReadCloser
//     that allows for a single read operation. This means that calling BindJSONOpt after the body has been read will likely result in an EOF error.
func (c *Context) BindJSONOpt(val any, useNumber bool, disableUnknown bool) error {
	if val == nil {
		return errs.ErrInputNil()
	}
	if c.Request.Body == nil {
		return errs.ErrBodyNil()
	}
	decoder := json.NewDecoder(c.Request.Body)
	if useNumber {
		decoder.UseNumber()
	}
	if disableUnknown {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(val)
}

// FormValue extracts a value from the form data in an HTTP request based on the given key.
// It parses the URL-encoded form data (both in the URL query string and the request body) and retrieves the value associated with the provided key.
//
// Parameters:
// - 'key' is a string representing the name of the form value to be retrieved from the HTTP request.
//
// The function performs the following actions:
//  1. It calls 'c.Request.ParseForm()' to parse the request body as a URL-encoded form. This is necessary to populate 'c.Request.Form' with the form data.
//     This method also parses the query parameters from the URL, merging them into the form values. If parsing fails (for example, if the body can't be read,
//     is too large, or if the content type is not application/x-www-form-urlencoded), an error is returned.
//  2. If 'ParseForm()' returns an error, the function creates a 'AnyValue' instance with an empty string for 'val' and the parsing error for 'err'.
//     It then returns this 'AnyValue' containing the error information.
//  3. If 'ParseForm()' succeeds, 'c.Request.FormValue(key)' is used to retrieve the first value for the specified key from the merged form data.
//     A 'AnyValue' instance is then returned with the retrieved value and 'nil' for the error.
//
// Return Value:
//   - A 'AnyValue' instance is always returned. This struct contains two fields:
//     a. 'Val' which is the any value retrieved from the form data.
//     b. 'Err' which captures any error that may have occurred during the parsing of the form data.
//
// Usage:
// - This method is intended to be used in HTTP handlers when you need to access form values sent in an HTTP request.
//
// Example:
// - To retrieve a form value named "email" from an HTTP request:
//
//	email := c.FormValue("email")
//	if email.err != nil {
//	   // handle the error (e.g., send an HTTP 400 Bad Request response)
//	}
//	// Use email.val as the required string value for "email".
//
// Note:
//   - The 'FormValue' method does not handle multiple values for the same key. It only retrieves the first such value.
//   - Calling 'FormValue' multiple times on the same request is safe as it does not reparse the form data.
//     The form data is parsed only once, and subsequent calls will retrieve values from the already parsed form.
//   - The 'ParseForm' method can only parse the request body if the method is "POST" or "PUT" and the content type is
//     "application/x-www-form-urlencoded". For other HTTP methods or content types, the body will not be parsed, but URL query parameters will still be available.
//
// Considerations:
// - Ensure that the 'ParseForm' method is not called before any other method that might consume the request body, as the request body is typically read-only once.
func (c *Context) FormValue(key string) AnyValue {
	err := c.Request.ParseForm()
	if err != nil {
		return AnyValue{
			Val: nil,
			Err: err,
		}
	}
	return AnyValue{Val: c.Request.FormValue(key)}
}

// QueryValue retrieves the first value associated with the specified key from the URL query parameters of an HTTP request.
//
// Parameters:
// - 'key' is the string that specifies the key in the query parameter that we want to retrieve the value for.
//
// This function operates as follows:
//  1. Checks if 'c.queryValues' is already populated. If it is not, it initializes it with the parsed query parameters obtained by calling 'c.Request.URL.Query()'.
//     This method parses the raw query from the URL and returns a map of the query parameters.
//  2. It looks for the given 'key' in 'c.queryValues' to see if it exists. The values are stored in a slice of strings because URL query parameters can have multiple values.
//  3. If the key does not exist in the map, it implies that the parameter was not supplied in the query string.
//     In this case, the function returns a 'AnyValue' struct with an empty string for the 'val' field and an 'ErrKeyNil' error for the 'err' field.
//  4. If the key is present, the function returns a 'AnyValue' struct with the first value associated with the key and 'nil' for the 'err' field.
//
// Return Value:
//   - A 'AnyValue' struct is always returned. It contains:
//     a. 'val', the value associated with the provided key from the query parameters.
//     b. 'err', an error if the key is not found in the query parameters.
//
// Usage:
// - The method is ideal when you need to obtain individual query parameters from an HTTP request without parsing the entire URL query string manually.
//
// Example:
// - To retrieve a query parameter named "page" from an HTTP request:
//
//	page := c.QueryValue("page")
//	if page.err != nil {
//	   // handle the error (e.g., the "page" query parameter was not provided)
//	}
//	// Use page.val to work with the value of "page".
//
// Note:
//   - This function only retrieves the first value for the specified key even if there are multiple values for that key.
//   - 'c.queryValues' is cached after its first use, which improves performance when accessing multiple query parameters
//     since it avoids reparsing the query string of the URL.
//   - It is important to correctly handle the error scenario since a missing key in the query parameters can affect application logic.
//
// Considerations:
// - While handling query data, consider the URL's sensitivity and the possibility of multiple values. Always validate and clean data from URL queries to ensure security.
func (c *Context) QueryValue(key string) AnyValue {
	if c.queryValues == nil {
		c.queryValues = c.Request.URL.Query()
	}

	vals, ok := c.queryValues[key]
	if !ok {
		return AnyValue{
			Val: nil,
			Err: errs.ErrKeyNil(),
		}
	}
	return AnyValue{Val: vals[0]}
}

// PathValue retrieves a value from the path parameters of an HTTP request based on the given key.
// The path parameters are typically extracted from the URL path, where they are defined by the routing patterns used in the application.
//
// Parameters:
// - 'key': A string representing the name of the path parameter to be retrieved.
//
// The function performs the following actions:
//  1. It checks if the map 'c.PathParams' contains the key provided in the method parameter.
//     'c.PathParams' is expected to be populated with key-value pairs where keys correspond to path parameter names defined in the URL pattern, and
//     values are the respective parameters extracted from the actual request URL.
//  2. If the key is present in 'c.PathParams', it retrieves the corresponding value, wraps it in a 'AnyValue' struct by setting the 'val' field to the retrieved
//     value and 'err' field to nil, and then returns it.
//  3. If the key is not found, it means the requested path parameter is not present in the request URL. In this case, the method returns a 'AnyValue' struct
//     with 'val' set to an empty string and 'err' set to an error instance that typically indicates the absence of the key.
//
// Return Value:
//   - A 'AnyValue' struct that contains the value associated with the provided key ('val') and an error field ('err').
//     If the key is not present, the 'err' field contains an appropriate error while 'val' is an empty string.
//
// Usage:
// - This method is useful in web server routing where URL paths may contain parameters that need to be extracted and used within the application logic.
//
// Example:
// - Assuming a URL pattern like "/users/:id" where :id is a path parameter:
//
//	userID := c.PathValue("id")
//	if userID.err != nil {
//	   // handle the error (e.g., send an HTTP 404 Not Found response)
//	}
//	// Use userID.val as the required user ID string value.
//
// Note:
//   - The method assumes that 'c.PathParams' has already been populated with the correct path parameters before calling 'PathValue'.
//     In a typical web server implementation, this population is done during the routing process, before the request handler is invoked.
//   - The 'PathParams' may hold multiple path parameters depending on the URL pattern; 'PathValue' method is responsible for extracting a single parameter by key.
//   - If the same key was present multiple times in 'c.PathParams', this method would return the first instance of the value associated with the key.
//
// Considerations:
//   - When working with frameworks or routers that facilitate path parameter extraction, ensure the router is correctly configured to parse and store the path parameters
//     before calling this method.
func (c *Context) PathValue(key string) AnyValue {
	val, ok := c.PathParams[key]
	if !ok {
		return AnyValue{
			Val: nil,
			Err: errs.ErrKeyNil(),
		}
	}
	return AnyValue{Val: val}
}

// Header sets or deletes a specific header in the HTTP response.
// If the given value is an empty string, the header is deleted.
// Otherwise, the value is set for the given key.
// Parameters:
// - key: the header name to set or delete.
// - value: the header value to set; if empty, the header with the given key is deleted.
func (c *Context) Header(key, value string) {
	if value == "" {
		// If the value is empty, delete the header from the response.
		c.ResponseWriter.Header().Del(key)
		return
	}
	// Set the header with the specified value.
	c.ResponseWriter.Header().Set(key, value)
}

// Set stores a value in the context under the specified key.
// This method is safe for concurrent use by multiple goroutines.
// Parameters:
// - key: the string key under which to store the value.
// - value: the value to be stored, which can be of any type.
func (c *Context) Set(key string, value any) {
	// Ensure exclusive access to the Keys map to prevent data races.
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Initialize the Keys map if it hasn't been created yet.
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}

	// Store the value under the specified key.
	c.Keys[key] = value
}

// Get retrieves a value stored in the context under the specified key.
// It returns the value and a boolean indicating whether the key exists in the map.
// This method is safe for concurrent use by multiple goroutines.
// Parameters:
// - key: the string key under which the value is stored.
// Returns:
// - value: the value stored under the key; nil if the key does not exist.
// - exists: a boolean indicating whether the key exists in the map.
func (c *Context) Get(key string) (value any, exists bool) {
	// Ensure access to the Keys map is synchronized to prevent data races.
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	// Retrieve the value and existence flag for the given key.
	value, exists = c.Keys[key]
	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Context) MustGet(key string) any {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// GetString retrieves a string value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - string: The string value associated with the key, or an empty string if the key does not exist or is not a string.
func (c *Context) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string) // Type assert the value to a string.
	}
	return
}

// GetBool retrieves a boolean value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - bool: The boolean value associated with the key, or false if the key does not exist or is not a boolean.
func (c *Context) GetBool(key string) (b bool) {
	if val, ok := c.Get(key); ok && val != nil {
		b, _ = val.(bool) // Type assert the value to a boolean.
	}
	return
}

// GetInt retrieves an integer value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - int: The integer value associated with the key, or 0 if the key does not exist or is not an integer.
func (c *Context) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.(int) // Type assert the value to an integer.
	}
	return
}

// GetInt64 retrieves an int64 value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - int64: The int64 value associated with the key, or 0 if the key does not exist or is not an int64.
func (c *Context) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64) // Type assert the value to an int64.
	}
	return
}

// GetUint retrieves an unsigned integer value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - uint: The unsigned integer value associated with the key, or 0 if the key does not exist or is not an unsigned integer.
func (c *Context) GetUint(key string) (ui uint) {
	if val, ok := c.Get(key); ok && val != nil {
		ui, _ = val.(uint) // Type assert the value to an unsigned integer.
	}
	return
}

// GetUint64 retrieves a uint64 value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - uint64: The uint64 value associated with the key, or 0 if the key does not exist or is not a uint64.
func (c *Context) GetUint64(key string) (ui64 uint64) {
	if val, ok := c.Get(key); ok && val != nil {
		ui64, _ = val.(uint64) // Type assert the value to a uint64.
	}
	return
}

// GetFloat64 retrieves a float64 value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - float64: The float64 value associated with the key, or 0 if the key does not exist or is not a float64.
func (c *Context) GetFloat64(key string) (f64 float64) {
	if val, ok := c.Get(key); ok && val != nil {
		f64, _ = val.(float64) // Type assert the value to a float64.
	}
	return
}

// GetTime retrieves a time.Time value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - time.Time: The time.Time value associated with the key, or the zero value of time.Time if the key does not exist or is not a time.Time.
func (c *Context) GetTime(key string) (t time.Time) {
	if val, ok := c.Get(key); ok && val != nil {
		t, _ = val.(time.Time) // Type assert the value to a time.Time.
	}
	return
}

// GetDuration retrieves a time.Duration value associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - time.Duration: The time.Duration value associated with the key, or 0 if the key does not exist or is not a time.Duration.
func (c *Context) GetDuration(key string) (d time.Duration) {
	if val, ok := c.Get(key); ok && val != nil {
		d, _ = val.(time.Duration) // Type assert the value to a time.Duration.
	}
	return
}

// GetStringSlice retrieves a slice of strings associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - []string: The slice of strings associated with the key, or nil if the key does not exist or is not a slice of strings.
func (c *Context) GetStringSlice(key string) (ss []string) {
	if val, ok := c.Get(key); ok && val != nil {
		ss, _ = val.([]string) // Type assert the value to a slice of strings.
	}
	return
}

// GetStringMap retrieves a map with string keys and any type values associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - map[string]any: The map associated with the key, or nil if the key does not exist or is not a map.
func (c *Context) GetStringMap(key string) (sm map[string]any) {
	if val, ok := c.Get(key); ok && val != nil {
		sm, _ = val.(map[string]any) // Type assert the value to a map with string keys and any type values.
	}
	return
}

// GetStringMapString retrieves a map with string keys and string values associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - map[string]string: The map associated with the key, or nil if the key does not exist or is not a map with string values.
func (c *Context) GetStringMapString(key string) (sms map[string]string) {
	if val, ok := c.Get(key); ok && val != nil {
		sms, _ = val.(map[string]string) // Type assert the value to a map with string keys and string values.
	}
	return
}

// GetStringMapStringSlice retrieves a map with string keys and slice of string values associated with the given key from the context.
// Parameters:
// - key: The key to retrieve the value for (string).
// Returns:
// - map[string][]string: The map associated with the key, or nil if the key does not exist or is not a map with slice of string values.
func (c *Context) GetStringMapStringSlice(key string) (smss map[string][]string) {
	if val, ok := c.Get(key); ok && val != nil {
		smss, _ = val.(map[string][]string) // Type assert the value to a map with string keys and slice of string values.
	}
	return
}
