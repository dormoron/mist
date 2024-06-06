package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// converter is a type alias for a function that converts a string to another string.
type converter func(string) string

// generateNormalHeaders generates headers for normal (non-preflight) CORS requests based on the configuration provided.
//
// Parameters:
// - c: The CORS configuration used to determine which headers to set.
//
// Returns:
// - A map of HTTP headers to be included in normal CORS responses.
func generateNormalHeaders(c Config) http.Header {
	headers := make(http.Header)

	// If credentials are allowed, set the corresponding header.
	if c.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	// If there are any expose headers, canonicalize them and set the appropriate header.
	if len(c.ExposeHeaders) > 0 {
		exposeHeaders := convert(normalize(c.ExposeHeaders), http.CanonicalHeaderKey)
		headers.Set("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ","))
	}

	// If all origins are allowed, set the corresponding header.
	if c.AllowAllOrigins {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		// Otherwise, vary the header by origin.
		headers.Set("Vary", "Origin")
	}

	return headers
}

// generatePreflightHeaders generates headers for preflight (OPTIONS) CORS requests based on the configuration provided.
//
// Parameters:
// - c: The CORS configuration used to determine which headers to set.
//
// Returns:
// - A map of HTTP headers to be included in preflight CORS responses.
func generatePreflightHeaders(c Config) http.Header {
	headers := make(http.Header)

	// If credentials are allowed, set the corresponding header.
	if c.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	// If there are any allowed methods, canonicalize them and set the appropriate header.
	if len(c.AllowMethods) > 0 {
		allowMethods := convert(normalize(c.AllowMethods), strings.ToUpper)
		value := strings.Join(allowMethods, ",")
		headers.Set("Access-Control-Allow-Methods", value)
	}

	// If there are any allowed headers, canonicalize them and set the appropriate header.
	if len(c.AllowHeaders) > 0 {
		allowHeaders := convert(normalize(c.AllowHeaders), http.CanonicalHeaderKey)
		value := strings.Join(allowHeaders, ",")
		headers.Set("Access-Control-Allow-Headers", value)
	}

	// If a max age is set, convert it to seconds and set the appropriate header.
	if c.MaxAge > time.Duration(0) {
		value := strconv.FormatInt(int64(c.MaxAge/time.Second), 10)
		headers.Set("Access-Control-Max-Age", value)
	}

	// If private networks are allowed, set the corresponding header.
	if c.AllowPrivateNetwork {
		headers.Set("Access-Control-Allow-Private-Network", "true")
	}

	// If all origins are allowed, set the corresponding header.
	if c.AllowAllOrigins {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		// Always set Vary headers to indicate that responses may vary based on these headers.
		headers.Add("Vary", "Origin")
		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")
	}

	return headers
}

// normalize takes a list of strings and returns a new list with:
// - Leading/trailing whitespace removed.
// - All strings lowercased.
// - Duplicates removed.
//
// Parameters:
// - values: A list of strings to be normalized.
//
// Returns:
// - A new list of normalized strings.
func normalize(values []string) []string {
	if values == nil {
		return nil
	}

	// Use a map to track distinct values.
	distinctMap := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))

	// Process each value in the input list.
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.ToLower(value)
		if _, seen := distinctMap[value]; !seen {
			normalized = append(normalized, value)
			distinctMap[value] = true
		}
	}

	return normalized
}

// convert takes a list of strings and applies a conversion function to each string.
// It returns a new list containing the converted strings.
//
// Parameters:
// - s: A list of strings to be converted.
// - c: A converter function that takes a string and returns a converted string.
//
// Returns:
// - A new list of converted strings.
func convert(s []string, c converter) []string {
	var out []string

	// Apply the converter function to each string in the input list.
	for _, i := range s {
		out = append(out, c(i))
	}

	return out
}
