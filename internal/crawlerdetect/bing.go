package crawlerdetect

// BingStrategy is a struct which embeds a pointer to an instance of the
// UniversalStrategy from the crawlerdetect package. This UniversalStrategy
// is designed to provide general mechanism for detection of web crawlers.
//
// Embedding the UniversalStrategy directly inside the BingStrategy struct
// allows it to inherit the methods and attributes of the UniversalStrategy,
// thereby enabling BingStrategy to act as a specialized version of the UniversalStrategy.
type BingStrategy struct {
	*UniversalStrategy
}

// InitBingStrategy function is responsible for the creation and initialization of
// a BingStrategy instance.
//
// Specifically, it creates a new BingStrategy and inside it, it initializes the
// embedded UniversalStrategy with a list of known hosts associated with a
// specific web crawler. In this case, the host "search.msn.com" is known to be
// associated with a web crawler from Microsoft's search engine, Bing.
//
// Returns:
//   - *BingStrategy: A pointer to an instance of the BingStrategy struct, with the
//     embedded UniversalStrategy initialized for detecting Bing's web crawler.
func InitBingStrategy() *BingStrategy {
	return &BingStrategy{
		UniversalStrategy: InitUniversalStrategy([]string{"search.msn.com"}),
	}
}
