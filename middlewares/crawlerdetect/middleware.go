package crawlerdetect

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/crawlerdetect"
	"log/slog"
	"net/http"
	"strings"
)

// Initialize constants for crawler identification strings as provided by the crawlerdetect package.
const Baidu = crawlerdetect.Baidu
const Bing = crawlerdetect.Bing
const Google = crawlerdetect.Google
const SoGou = crawlerdetect.SoGou

// Define the MiddlewareBuilder struct to configure middleware for detecting web crawlers.
type MiddlewareBuilder struct {
	// crawlersMap holds the association of crawler user agent strings to the crawler names.
	crawlersMap map[string]string
}

// InitMiddlewareBuilder initializes and returns an instance of MiddlewareBuilder.
// It sets the default mappings between user agent strings of crawlers and their
// respective names from the crawlerdetect package.
func InitMiddlewareBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		crawlersMap: map[string]string{
			// Initialize the map with pairs of user agent strings and crawler names.
			// Baidu crawlers:
			"Baiduspider":        Baidu,
			"Baiduspider-render": Baidu,

			// Bing crawlers:
			"bingbot":          Bing,
			"adidxbot":         Bing,
			"MicrosoftPreview": Bing,

			// Google crawlers:
			"Googlebot":             Google,
			"Googlebot-Image":       Google,
			"Googlebot-News":        Google,
			"Googlebot-Video":       Google,
			"Storebot-Google":       Google,
			"Google-InspectionTool": Google,
			"GoogleOther":           Google,
			"Google-Extended":       Google,

			// SoGou crawlers:
			"Sogou web spider": SoGou,
		},
	}
}

// AddUserAgent takes a map of crawlers to their corresponding user agents (as slices of strings)
// and updates the MiddlewareBuilder's crawlersMap to include these associations.
// Parameters:
//   - userAgents: a map where the key is a string representing the crawler's name,
//     and the value is a slice of strings representing the user agent strings associated with that crawler.
//
// Returns:
// - a pointer to the updated MiddlewareBuilder for method chaining.
func (b *MiddlewareBuilder) AddUserAgent(userAgents map[string][]string) *MiddlewareBuilder {
	for crawler, values := range userAgents {
		// Iterate over each user agent string in the slice for a given crawler.
		for _, userAgent := range values {
			// Add or update the crawler's map to associate the user agent string with the crawler name.
			b.crawlersMap[userAgent] = crawler
		}
	}
	// Return the MiddlewareBuilder pointer to allow for method chaining.
	return b
}

// RemoveUserAgent takes a variadic parameter of user agent strings and removes them from
// the MiddlewareBuilder's crawlersMap if they exist.
// Parameters:
//   - userAgents: a variadic parameter where each argument is a string representing
//     a user agent to be removed from the MiddlewareBuilder's crawlersMap.
//
// Returns:
// - a pointer to the updated MiddlewareBuilder for method chaining.
func (b *MiddlewareBuilder) RemoveUserAgent(userAgents ...string) *MiddlewareBuilder {
	for _, userAgent := range userAgents {
		// Delete the entry for each user agent string provided from the crawlersMap.
		delete(b.crawlersMap, userAgent)
	}
	// Return the MiddlewareBuilder pointer to allow for method chaining.
	return b
}

// Build creates a new middleware function that intercepts HTTP requests to perform
// crawler detection based on the user agent and client IP address.
// Returns:
// - a middleware function that fits the mist.Middleware function signature.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	// Return a function that takes the next handler in the middleware chain.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Return the actual middleware function to be executed in the middleware chain.
		return func(ctx *mist.Context) {
			// Retrieve the User-Agent header from the incoming HTTP request.
			userAgent := ctx.Request.Header.Get("User-Agent")
			// Retrieve the client's IP address.
			ip := ctx.ClientIP()
			// If the IP is empty, log an error and abort the request with a 403 (Forbidden) status.
			if ip == "" {
				slog.ErrorContext(ctx.Request.Context(), "crawlerdetect", "error", "ip is empty.")
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			// Get the appropriate crawler detector based on the user agent string.
			crawlerDetector := b.getCrawlerDetector(userAgent)
			// If no detector is found, abort the request with a 403 (Forbidden) status.
			if crawlerDetector == nil {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			// Use the detector to check if the IP address belongs to a crawler.
			pass, err := crawlerDetector.CheckCrawler(ip)
			// If there is an error during the check, log it and abort with a 500 (Internal Server Error) status.
			if err != nil {
				slog.ErrorContext(ctx.Request.Context(), "crawlerdetect", "error", err.Error())
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			// If the IP is identified as a crawler, abort the request with a 403 (Forbidden) status.
			if !pass {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			// If the request passes the crawler check, proceed to the next handler in the middleware chain.
			next(ctx)
		}
	}
}

// getCrawlerDetector is a helper function that matches a given user agent string
// against the keys in the MiddlewareBuilder's crawlersMap. If a match is found,
// it initializes and returns a corresponding crawler detection strategy.
// Parameters:
// - userAgent: a string representing the user agent to be checked.
// Returns:
// - a crawler detection strategy if a matching user agent key is found in the crawlersMap, or nil otherwise.
func (b *MiddlewareBuilder) getCrawlerDetector(userAgent string) crawlerdetect.Strategy {
	for key, value := range b.crawlersMap {
		// Check if the user agent string contains the current key from the map.
		if strings.Contains(userAgent, key) {
			// If a match is found, initialize a crawler detector with the matched crawler name and return it.
			return crawlerdetect.InitCrawlerDetector(value)
		}
	}
	// If no match is found after checking all keys in the map, return nil.
	return nil
}
