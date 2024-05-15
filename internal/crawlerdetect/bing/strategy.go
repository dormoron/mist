package bing

import "github.com/dormoron/mist/internal/crawlerdetect"

// Strategy is a struct which embeds a pointer to an instance of the
// UniversalStrategy from the crawlerdetect package. This UniversalStrategy
// is designed to provide general mechanism for detection of web crawlers.
//
// Embedding the UniversalStrategy directly inside the Strategy struct
// allows it to inherit the methods and attributes of the UniversalStrategy,
// thereby enabling Strategy to act as a specialized version of the UniversalStrategy.
type Strategy struct {
	*crawlerdetect.UniversalStrategy
}

// InitStrategy function is responsible for the creation and initialization of
// a Strategy instance.
//
// Specifically, it creates a new Strategy and inside it, it initializes the
// embedded UniversalStrategy with a list of known hosts associated with a
// specific web crawler. In this case, the host "search.msn.com" is known to be
// associated with a web crawler from Microsoft's search engine, Bing.
//
// Returns:
//   - *Strategy: A pointer to an instance of the Strategy struct, with the
//     embedded UniversalStrategy initialized for detecting Bing's web crawler.
func InitStrategy() *Strategy {
	return &Strategy{
		UniversalStrategy: crawlerdetect.InitUniversalStrategy([]string{"search.msn.com"}),
	}
}
