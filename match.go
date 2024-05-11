package mist

// matchInfo holds the necessary information for a matched route. It encapsulates the node that has been matched,
// any path parameters extracted from the URL, and a list of middleware that should be applied for the route.
// This struct is typically used in the context of a routing system, where it is responsible for carrying the
// cumulative data required to handle an HTTP request after a route has been successfully matched.
//
// Fields:
//   - n (*node): A pointer to the matched 'node' which represents the endpoint in the routing tree that has been
//     matched against the incoming request path. This 'node' contains the necessary information to process
//     the request further, such as associated handlers or additional routing information.
//   - pathParams (map[string]string): A map that stores the path parameters as key-value pairs, where the key is
//     the name of the parameter (as defined in the path) and the value is the actual
//     string that has been matched from the request URL. For example, for a route
//     pattern "/users/:userID/posts/:postID", this map would contain entries for
//     "userID" and "postID" if the incoming request path matched that pattern.
//   - mils ([]Middleware): A slice of 'Middleware' functions that are meant to be executed for the matched route
//     in the order they are included in the slice. Middleware functions are used to perform
//     operations such as request logging, authentication, and input validation before the
//     request reaches the final handler function.
//
// Usage:
// The 'matchInfo' struct is populated during the route-matching process. Once a request path is matched against
// the routing tree, a 'matchInfo' instance is created and filled with the corresponding node, extracted path
// parameters, and any middleware associated with the matched route. This instance is then passed along to the
// request handling logic, where it guides the processing of the request through various middleware layers and
// eventually to the appropriate handler that will generate the response.
type matchInfo struct {
	// n is the node corresponding to the matched route in the routing tree. It provides access to any additional
	// route-specific information required to handle the request.
	n *node

	// pathParams stores the parameters identified in the URL path, such as "id" in "/users/:id", mapped to their
	// actual values as resolved from the incoming request.
	pathParams map[string]string

	// mils is a collection of middleware functions to be executed sequentially for the matched route. These functions
	// can modify the request context, perform checks, or carry out other pre-processing tasks.
	mils []Middleware
}

// addValue is a method that adds a key-value pair to the pathParams map of the matchInfo struct. This method
// serves to accumulate the parameters extracted from a matched URL path and store them for later use during
// the request-handling process.
//
// Parameters:
//   - key: A string representing the name of the URL parameter (e.g., "userID").
//   - value: A string representing the value of the URL parameter that has been extracted from the request
//     URL (e.g., "42" for a userID).
//
// The addValue function performs these steps:
//
//  1. Checks if the pathParams map inside the matchInfo struct is nil, which would indicate that no parameters
//     have been added yet. If it is nil, it initializes the pathParams map and instantly adds the key-value
//     pair to it. This is necessary because you cannot add keys to a nil map; it must be initialized first.
//  2. If the pathParams map is already initialized, it adds or overwrites the entry for the key with the new value.
//     This ensures that the most recently processed value for a given key is stored in the map.
//
// Usage:
// The addValue method is typically called during the route matching process, where path segments corresponding
// to parameters in the route pattern are parsed and their values accumulated. Each time a segment is processed
// and a parameter value is extracted, addValue is used to save that value with the corresponding parameter name.
//
// Example:
// For a URL pattern like "/users/:userID", when processing a request path like "/users/42", the method would
// be invoked as addValue("userID", "42"), adding the parameter "userID" with the value "42" to the pathParams map.
func (m *matchInfo) addValue(key string, value string) {
	// Initialize the pathParams map if it hasn't been already to avoid nil map assignment panic.
	if m.pathParams == nil {
		m.pathParams = map[string]string{key: value}
	}
	// Add or update the pathParams map with the key-value pair representing the URL parameter and its value.
	m.pathParams[key] = value
}
