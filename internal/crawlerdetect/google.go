package crawlerdetect

// GoogleStrategy is a struct that embeds a pointer to a UniversalStrategy from the
// crawler detect package. This is a pattern often used in Go to achieve composition,
// where GoogleStrategy 'is-a' UniversalStrategy, gaining access to its methods and properties directly.
// The purpose of embedding this specific UniversalStrategy is to leverage predefined methods
// and capabilities for detecting web crawlers based on a list of hosts.
type GoogleStrategy struct {
	*UniversalStrategy
}

// InitGoogleStrategy is a function that initializes and returns a pointer to a GoogleStrategy instance.
// It specifically initializes the embedded UniversalStrategy field with a set of hosts
// that are known to be associated with Google's web crawlers. This setup is useful for
// systems looking to detect and possibly differentiate traffic originating from Google's crawlers.
//
// Returns:
//   - *GoogleStrategy: A pointer to the newly created GoogleStrategy instance. This instance now contains a
//     UniversalStrategy initialized with a predefined list of hosts known to be used by Google's crawlers.
//
// Usage Notes:
//   - The list of hosts ('googlebot.com', 'google.com', 'googleusercontent.com') are specifically
//     chosen because they are commonly associated with Google's web crawling services. The intention
//     is to recognize traffic from these entities during web crawling detection checks.
//   - This setup is particularly useful for SEO-sensitive websites or web applications that might
//     want to tailor their responses based on whether the traffic is generated by human users or
//     automated crawlers.
func InitGoogleStrategy() *GoogleStrategy {
	return &GoogleStrategy{
		UniversalStrategy: InitUniversalStrategy([]string{"googlebot.com", "google.com", "googleusercontent.com"}),
	}
}
