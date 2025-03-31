package mist

import (
	"regexp"
	"strings"

	"github.com/dormoron/mist/internal/errs"
)

// Enumeration of node types for structuring route segments within the routing tree. Each constant represents a
// specific kind of route node and dictates how match operations should be conducted for the segment of the path
// it represents. The node types are defined using iota for incrementing integer values starting from 0, which
// provides a unique identifier for each node type.
//
// nodeTypeStatic:
// Used to represent nodes with static path segments. A static node is one that matches exactly with the path segment.
// For example, in the path "/api/books", 'api' and 'books' represent static segments. These segments must be
// present and identical in the request path for a match to occur. Static nodes are the most common and are used
// for fixed-path routing.
//
// nodeTypeReg:
// This type is used for nodes that should match a regular expression pattern. It allows more complex and flexible
// matching beyond static equality. Such nodes allow the matching of segments that conform to a specific pattern
// defined by a regular expression.
//
// nodeTypeParam:
// Used to represent nodes with parameterized path segments. Parameterized segments capture dynamic values. These
// nodes often start with a colon (':') followed by the parameter name in the path pattern (e.g., "/books/:id").
// The actual path segment in the request URL at this position will be captured as a named parameter that can be
// used within the application (e.g., to retrieve a book by its 'id' from a database).
//
// nodeTypeAny:
// Represents nodes that are intended to match any path segment(s). It often symbolizes wildcard or catch-all
// segments in routing, which can be used to capture all remaining path information. For instance, a pattern like
// "/files/*" with a nodeTypeAny node can match any subsequent path elements after "/files/", allowing flexibility
// in handling requests for a variable-depth file directory structure.
//
// These constants are integral to the route matching logic within the routing system. They guide the router when
// determining whether a given route node matches a segment of the request URL and whether to process it as a static
// value, a pattern, a parameter, or a wildcard segment.
const (
	nodeTypeStatic = iota // Indicates a node matches a specific and unchanging route segment.
	nodeTypeReg           // Indicates a node matches a route segment based on a regular expression.
	nodeTypeParam         // Indicates a node represents a named parameter within a route segment.
	nodeTypeAny           // Indicates a node is a wildcard, matching any sequence of route segments.
)

// nodeType is an enumerated type (int) used to categorize the different kinds of nodes that can exist within the
// routing structure of a router. Each node in the routing hierarchy represents a segment of a route's path,
// and the type of node can affect route matching behavior and the way in which parameters are extracted from
// the path during route resolution.
//
// A route's path can be composed of fixed, parameterized, or wildcard segments, and the nodeType helps to
// distinguish between these possibilities. Fixed segments match exactly, parameterized segments capture path
// variables, and wildcard nodes can match any segment or sequence of segments.
//
// This custom type enhances type safety by restricting the set of values that can represent node types to those
// explicitly defined in the associated constant definitions that follow the type declaration. This enables
// compile-time checks for the values of nodeType, ensuring that only valid types are used within the routing logic.
//
// Usage of nodeType:
//   - The nodeType is used by internal routing structures to manage and process different kinds of route segments.
//   - It is used within switch statements or conditional blocks when processing incoming paths to determine how to
//     match a given segment and how to proceed with traversal or parameter extraction.
//
// Declaration of nodeType constants:
// Following the type definition, constants are typically declared to represent the allowable nodeType values.
// For example:
//
//	const (
//	    staticNodeType nodeType = iota // Represents a node with a static segment.
//	    paramNodeType                  // Represents a node with a parameterized segment.
//	    wildcardNodeType               // Represents a node with a wildcard segment.
//	)
//
// - `staticNodeType` would be used for routes with fixed paths like "/books".
// - `paramNodeType` would be used for dynamically parameterized paths like "/books/:id" where ":id" is a parameter.
// - `wildcardNodeType` might be used for routes that should match any remaining path like "/files/*filepath".
//
// With these constants, the developer can then work with nodes of specific types without worrying about using
// integer literals throughout the code, enhancing readability, and reducing the potential for errors.
type nodeType int

// node represents a segment within the URL path hierarchy of a routing structure.
// Each node can represent a static path segment, a parameter (dynamic segment), or a
// wildcard, and can hold additional information like handlers and middleware necessary for routing.
//
// Fields:
// - typ: nodeType indicates the type of the node (e.g., static, parameter, wildcard).
//
//   - route: A string that captures the full route pattern this node is part of,
//     which could be helpful for debugging or route listing features.
//
// - path: The specific segment of the route that this node represents.
//
//   - children: A map where keys are path segments and values are pointers to child nodes,
//     allowing the representation of a hierarchical routing structure.
//
//   - handler: The HandleFunc to invoke when this node's route is matched. It contains the logic to
//     handle the incoming request for the associated route.
//
//   - starChild: A pointer to a child node that represents a wildcard segment, capturing any text in
//     a path segment where '*” has been used within the route pattern.
//
//   - paramChild: A pointer to a child node that specifies a route parameter segment, such as ':id',
//     which captures a named variable from the path.
//
// - paramName: The name of the parameter captured by this node if it's a paramChild (e.g., 'id' from ':id').
//
//   - mils: A slice of Middleware functions that are associated with the node, to be executed
//     before the handler when the route is matched.
//
//   - matchedMils: A slice of Middleware that were matched and need to be invoked for the current
//     route. This can be built up as the route is resolved.
//
//   - regChild: A pointer to a child node that represents a regular expression pattern segment,
//     which captures complex variable patterns from the path.
//
//   - regExpr: The compiled regular expression (if applicable) that matches the route segment associated
//     with this node.
//
// - parent: A pointer to the parent node in the routing hierarchy, allowing traversal to the root.
//
// Usage:
// The node structure is typically used within the implementation of a router or a middleware
// to build a hierarchical representation of the application's routes. Each route in the application
// corresponds to a chain of nodes, from the root node to the leaf node representing the endpoint.
type node struct {
	typ         nodeType
	route       string
	path        string
	children    map[string]*node
	handler     HandleFunc
	starChild   *node
	paramChild  *node
	paramName   string
	mils        []Middleware
	matchedMils []Middleware
	regChild    *node
	regExpr     *regexp.Regexp
	parent      *node
}

// childrenOf searches through the current node's children to construct a slice of child nodes that match or relate to the given path segment.
// The method considers static (exact match), parameterized, and wildcard children and includes them in the result as they represent possible
// routes for a given path in a routing hierarchy. This is useful in situations where multiple routes could handle the same path, such as in web
// frameworks that support route parameters and wildcards.
//
// Parameters:
// - path: A string representing the path segment to search for among the current node's children.
//
// Returns:
//   - []*node: A slice of pointers to node objects that are children of the current node and correspond to the path segment provided. This slice
//     contains wildcard and parameterized children, with static children (if any match exists) appended last.
func (n *node) childrenOf(path string) []*node {
	// Initialize a slice of node pointers with an initial capacity to store potential children.
	res := make([]*node, 0, 4)

	// Declare a variable to hold a static child node if it exists.
	var static *node

	// Check if the current node has children and attempt to find a static child that matches the path.
	if n.children != nil {
		static = n.children[path]
	}

	// If the current node has a wildcard child, append it to the result slice.
	if n.starChild != nil {
		res = append(res, n.starChild)
	}

	// If the current node has a parameterized child, append it to the result slice.
	if n.paramChild != nil {
		res = append(res, n.paramChild)
	}

	// If a static child exists, append it to the result slice after wildcard and parameterized children.
	if static != nil {
		res = append(res, static)
	}

	// Return the populated slice of child nodes.
	return res
}

// childOf attempts to retrieve a child node associated with a given path segment from the current node.
// It differentiates between exact path matches, parameterized path segments, and wildcard path segments.
// This method is usually called during the traversal of a routing tree in a web framework or any nested
// data structure where nodes may represent different parts of a hierarchical path, such as a filesystem.
//
// Parameters:
// - path: A string representing the exact path segment to match against the node's children.
//
// Returns:
//   - *node: A pointer to the child node that matches the path segment. If there is no exact match, it returns
//     the parameterized or wildcard child node. If no matches are found, it returns nil.
//   - bool: A boolean value that indicates if the returned node is a parameterized child node. It is true if the
//     result is from a parameterized path segment, false otherwise.
//   - bool: A boolean value which indicates whether a successful match was found. It is true if either an exact match,
//     parameterized match, or wildcard match is found, false if there is no child node for the path segment.
func (n *node) childOf(path string) (*node, bool, bool) {
	// If the current node does not have any children nodes, check for parameterized or wildcard child nodes.
	if n.children == nil {
		// If a parameterized child exists, return it along with true for both boolean values, indicating a match and parameterized match.
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		// If only a star child exists (wildcard node), return it with false for parameterized match but true to indicate a match was found.
		return n.starChild, false, n.starChild != nil
	}

	// Attempt to find an exact match for the path in the children node map.
	res, ok := n.children[path]
	if !ok {
		// If no exact match is found, check again for parameterized or wildcard children, similar to the logic above.
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		return n.starChild, false, n.starChild != nil
	}

	// If an exact match is found, return it along with false for both boolean values, indicating an exact match without any parameterization.
	return res, false, ok
}

// childOfNonStatic attempts to find a non-static (dynamic) child node of the current node (n) that matches the given
// path segment. This includes children nodes that represent regular expression patterns, named parameters, or wildcard
// segments. It returns a pointer to the matching child node and a boolean flag indicating whether a match was found.
//
// Parameters:
// - path: A string representing the path segment to match against the current node's dynamic children.
//
// The childOfNonStatic function operates in the following sequence:
//
//  1. Checks if the current node has a regular expression child (regChild). If so, it uses the compiled regular
//     expression stored in regChild.regExpr to determine if the given path segment matches the pattern.
//  2. If a match is confirmed with the regular expression, the regChild node and 'true' are returned to indicate
//     successful matching.
//  3. If there is no regChild or if the path does not match the regular expression, the function then checks whether
//     the current node has a parameterized child (paramChild). Parameterized children represent path segments with
//     named parameters (e.g., /users/:userId).
//  4. If a paramChild exists, it is assumed to match the path segment (since parameterized segments can match any
//     value), and the paramChild node and 'true' are returned.
//  5. If neither a regChild nor a paramChild are applicable, the function finally checks for the presence of a wildcard
//     child (starChild). Wildcard children are used to match any remaining path segments, typically represented by an
//     asterisk (*).
//  6. If a starChild exists, it is returned along with 'true', as it matches any path by definition. If starChild does
//     not exist, the function returns nil and 'false', meaning no match was found among the node's dynamic children.
//
// This method is specifically designed to handle dynamic routing scenarios where path segments may not be known
// statically and can contain patterns, parameters, or wildcards that need to be resolved at runtime.
func (n *node) childOfNonStatic(path string) (*node, bool) {
	// Attempt to match the path segment with a regular expression pattern if regChild exists.
	if n.regChild != nil {
		// If the regular expression matches the path, return the regChild and true.
		if n.regChild.regExpr.Match([]byte(path)) {
			return n.regChild, true
		}
	}

	// If no regular expression match is found, check for a parameterized child node.
	if n.paramChild != nil {
		// Parameterized child nodes match any path segment, so return the paramChild and true.
		return n.paramChild, true
	}

	// If no other dynamic match is found, check for a wildcard child node.
	// Wildcard nodes (if any) match any path segment, so return starChild and a boolean indicating its existence.
	return n.starChild, n.starChild != nil
}

// childOrCreate locates a child node within the current node (n) that matches the given 'path' or creates a new
// child node if one does not already exist. It handles different node types: static, parameterized, regular
// expression-based, and wildcard. The method returns a pointer to the child node. If the given 'path' represents a
// wildcard or parameterized path and violates the routing rules (such as being mixed with parameterized paths or
// regular expressions), the method panics with an appropriate error.
//
// Parameters:
// - path: A string representing the path segment to match against or to create within the current node's children.
//
// The childOrCreate function operates as follows:
//
//  1. Checks if the given 'path' is a wildcard "*". If so, it ensures that no parameterized (paramChild) or regular
//     expression-based (regChild) children exist, as these are not allowed in conjunction with a wildcard. If this
//     rule is violated, a panic occurs with a descriptive error message.
//  2. If a wildcard child node does not exist, it creates one, initializes it with the path, and sets its type to
//     nodeTypeAny.
//  3. If the given 'path' starts with ':', indicating it is a parameterized path, the method parses the parameter name
//     and any associated regular expression (if present) using 'parseParam'.
//  4. Depending on whether a regular expression is part of the parameterized path, it calls either 'childOrCreateReg'
//     or 'childOrCreateParam' to either create or fetch the existing child.
//  5. If 'path' does not start with '*' or ':', indicating a static path, it initializes 'n.children' if it's nil and
//     then looks for or creates a static child node with the given path.
//  6. It inserts the new static child node into the 'children' map if it does not exist already and initializes it
//     with the path and type 'nodeTypeStatic'.
//
// Note:
// - This method modifies the current node 'n', potentially adding new child nodes to it.
// - This method assumes that 'path' is a non-empty string.
func (n *node) childOrCreate(path string) *node {
	// Wildcard path handling: creates or retrieves a wildcard child, enforcing rules against mixing wildcard
	// with parameter and regular expression children.
	if path == "*" {
		// Check and enforce routing rule: Wildcards cannot exist alongside parameterized paths.
		if n.paramChild != nil {
			panic(errs.ErrPathNotAllowWildcardAndPath(path))
		}
		// Check and enforce routing rule: Wildcards cannot exist alongside regular expression paths.
		if n.regChild != nil {
			panic(errs.ErrRegularNotAllowWildcardAndRegular(path))
		}
		// Create a wildcard child node if one does not exist, initialize and store it for future retrievals.
		if n.starChild == nil {
			n.starChild = &node{
				path:   path,
				typ:    nodeTypeAny,
				parent: n, // 设置父节点关系
			}
		}
		return n.starChild // Return the wildcard child node.
	}

	// 支持{name:regex}格式的正则表达式 - 大括号格式
	if len(path) > 3 && path[0] == '{' && strings.Contains(path, ":") {
		closeBrace := strings.LastIndex(path, "}")
		if closeBrace != -1 && closeBrace > 2 {
			parts := strings.SplitN(path[1:closeBrace], ":", 2)
			if len(parts) == 2 {
				paramName := parts[0]
				expr := parts[1]
				return n.childOrCreateReg(path, expr, paramName)
			}
		}
		// 如果格式不正确，抛出异常
		panic(errs.ErrInvalidRegularFormat(path))
	}

	// Parameterized path handling: parses the parameter name and expression, and creates or retrieves
	// the corresponding parameterized or regular expression child node.
	if path[0] == ':' {
		paramName, expr, isReg := n.parseParam(path)
		if isReg {
			// For paths with an embedded regular expression, create or retrieve a regular expression child node.
			return n.childOrCreateReg(path, expr, paramName)
		}
		// For standard parameterized paths, create or retrieve a parameterized child node.
		return n.childOrCreateParam(path, paramName)
	}

	// Static path handling: creates or retrieves a static child node.
	if n.children == nil {
		// Initialize the children map if it hasn't been already to prevent nil map assignment errors.
		n.children = make(map[string]*node)
	}
	// Look for or create a static child node for the given path.
	child, ok := n.children[path]
	if !ok {
		// If the child node does not exist already, create it, initialize it with the path and type,
		// and add it to the children map.
		child = &node{
			path:   path,
			typ:    nodeTypeStatic,
			parent: n, // 设置父节点关系
		}
		n.children[path] = child
	}
	return child // Return the static child node.
}

// childOrCreateParam is used to retrieve an existing or create a new parameterized child node associated with the current
// node (n). It manages nodes that represent path parameters in a URL, usually denoted by a colon (':') followed by the
// parameter name (e.g., ":id" in "/users/:id"). The method ensures that parameter nodes do not coexist with wildcard or
// regular expression nodes, as per routing rules. It panics if a routing conflict occurs.
//
// Parameters:
// - path: The path segment that the method attempts to match or create a node for.
// - paramName: The name of the parameter as extracted from the path.
//
// The childOrCreateParam function performs the following actions:
//
//  1. First, it checks if the current node has a child that is a regular expression node (regChild). If such a child
//     exists, it's considered a routing conflict because a regular expression child cannot coexist with a parameterized
//     path. In this case, the method panics with an appropriate error.
//  2. Next, it checks for the presence of a wildcard child (starChild). Again, as per routing rules, a wildcard child
//     cannot coexist with a parameterized child, and if found, the method panics with an error.
//  3. The method then checks if a parameterized child node (paramChild) already exists. If it does and its path differs
//     from the given 'path', this is considered a routing conflict (two different parameterized paths cannot be the same
//     route segment), prompting the method to panic with a path clash error.
//  4. If no parameterized child exists or if the existing one has the same path, the method is safe to proceed. If a new
//     child needs to be created, it's initialized with the given 'path', 'paramName', and set to nodeTypeParam to
//     denote its nature as a parameterized node.
//  5. Finally, the existing or newly created parameterized child node is returned.
//
// Note:
// - This method updates the current node 'n' by potentially adding a paramChild.
// - It only handles parameterized paths and is part of a broader routing system with rules to prevent routing conflicts.
func (n *node) childOrCreateParam(path string, paramName string) *node {
	// Enforce routing rules by checking for the presence of regular expression and wildcard children,
	// and panic if necessary to prevent invalid routing configurations.
	if n.regChild != nil {
		panic(errs.ErrRegularNotAllowRegularAndPath(path))
	}
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndPath(path))
	}
	// Check if a parameterized child node already exists with the same path.
	if n.paramChild != nil {
		// If the paths differ, this denotes a routing conflict, and panic with an error.
		if n.paramChild.path != path {
			panic(errs.ErrPathClash(n.paramChild.path, path))
		}
	} else {
		// If no parameterized child exists, create one with the provided path and parameter name.
		n.paramChild = &node{
			path:      path,
			paramName: paramName,
			typ:       nodeTypeParam,
			parent:    n, // 设置父节点关系
		}
	}
	// Return the existing or newly created parameterized child node.
	return n.paramChild
}

// childOrCreateReg retrieves or creates a child node that represents a path segment with an embedded regular
// expression. This method is called when the path segment includes a parameter with a custom regular expression
// constraint, denoting a more complex matching requirement than standard parameterized routes.
//
// Parameters:
// - path: The full path segment including the parameter and its associated regular expression (e.g., ":id(\\d+)").
// - expr: The raw regular expression string used to match this parameter (e.g., "\\d+").
// - paramName: The name of the parameter to be extracted from the path (e.g., "id").
//
// The childOrCreateReg function performs these steps:
//
//  1. It ensures that no wildcard child (starChild) exists, as mixing wildcards with regular expression constrained
//     parameters is not permissible. If a wildcard is present, the function panics with the appropriate error.
//  2. It ensures that no simple parameterized child (paramChild) exists, as such nodes cannot coexist with regular
//     expression constrained parameters. If found, the function panics with a relevant error message.
//  3. If a regular expression child (regChild) already exists, the method checks that its regular expression and
//     parameter name match the current ones. If they do not, indicating a clash in the routing definitions, the
//     method panics with a routing conflict error.
//  4. If no regChild exists that meets the required criteria, the method creates one. This involves compiling the
//     passed regular expression to create a 'regexp.Regexp' object. If compiling fails, it panics with an error
//     that indicates an issue with the regular expression.
//  5. Finally, the new or existing regular expression child node is returned.
//
// Note:
//   - The method updates the current 'node' by adding a regChild if necessary.
//   - It only manages nodes with regular expression constraints and upholds routing system integrity by checking for
//     potential routing definition clashes.
func (n *node) childOrCreateReg(path string, expr string, paramName string) *node {
	// Check for and enforce routing conflicts with wildcard and param nodes. Panic if a conflict exists.
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndRegular(path))
	}
	if n.paramChild != nil {
		panic(errs.ErrPathNotAllowPathAndRegular(path))
	}
	// If a regular expression child already exists, ensure it matches the new requirements. Otherwise, panic.
	if n.regChild != nil {
		// A routing definition clash occurs when the existing regChild's regular expression or parameter name
		// does not match the new requirements. Panic with an error indicating this conflict.
		if (n.regChild.regExpr != nil && n.regChild.regExpr.String() != expr) || n.regChild.paramName != paramName {
			panic(errs.ErrRegularClash(n.regChild.path, path))
		}
	} else {
		// Compile the new regular expression, and panic with an error if there's an issue with the compilation.
		regExpr, err := regexp.Compile(expr)
		if err != nil {
			panic(errs.ErrRegularExpression(err))
		}
		// If successful, create a new regChild node with the compiled expression and other data, and assign it to the current node.
		n.regChild = &node{
			path:      path,
			paramName: paramName,
			regExpr:   regExpr,
			typ:       nodeTypeReg,
			parent:    n, // 设置父节点关系
		}
	}
	// Return the existing or newly created regChild node.
	return n.regChild
}

// parseParam analyzes a given path segment to identify and extract the name of the parameter and, optionally,
// any regular expression associated with it. This is used in routing to handle dynamic segments in URLs. The
// method returns a tuple with the parameter name, the extracted regular expression (if any), and a boolean
// indicating whether a regular expression was found.
//
// Parameters:
//   - path: A string representing the segment of the URL path that contains the parameter. This should start with
//     a colon (':') followed by the parameter name and may include an embedded regular expression.
//
// The parseParam function proceeds as follows:
//
//  1. Removes the leading colon (':') from the path segment, as it only serves as an identifier of a parameter segment.
//  2. Splits the remaining string into two parts at the first occurrence of an opening parenthesis '(' which would
//     indicate the start of a regular expression constraint on the parameter.
//  3. If the split result has two segments, then a regular expression is assumed to be present:
//     - It further checks if the second segment has a closing parenthesis ')'. This confirms a well-formed regular
//     expression constraint. If it is well-formed, the regular expression is extracted, excluding the parentheses.
//     - It returns the parameter name, the regular expression without the enclosing parentheses, and true (for the
//     boolean indicating the presence of a regular expression).
//  4. If no regular expression is found or the regular expression is not well-formed (e.g., missing the closing
//     parenthesis or not having any parentheses at all), it returns the parameter name as the whole path after
//     the colon, an empty string for the regular expression, and false (no regular expression was found).
//
// Note:
//   - This method is utilized when building the routing tree to recognize and correctly process different node types
//     based on their path definitions.
//   - It is crucial for ensuring that URL parameters can be correctly matched and extracted during request handling.
func (n *node) parseParam(path string) (string, string, bool) {
	// Remove the leading colon from the path to isolate the parameter name and potential regular expression.
	path = path[1:]
	// Attempt to split the path segment at the opening parenthesis to separate the parameter name from any regular expression.
	segs := strings.SplitN(path, "(", 2)
	// Check if a regular expression is present by seeing if there are two segments after the split.
	if len(segs) == 2 {
		// Assuming the second segment is a regular expression, check if it ends with a closing parenthesis.
		expr := segs[1]
		if strings.HasSuffix(expr, ")") {
			// If so, return the parameter name, the regular expression without parentheses, and true.
			return segs[0], expr[:len(expr)-1], true
		}
	}
	// If there is no regular expression, return the parameter name, an empty string, and false.
	return path, "", false
}
