package baidu

import "github.com/dormoron/mist/internal/crawlerdetect"

// Strategy is a struct used for detecting web crawlers. It embeds a pointer
// to an instance of the UniversalStrategy from the crawlerdetect package.
// By embedding it, the Strategy struct can directly call the methods
// and use the properties of the `UniversalStrategy`, achieving behavior akin to inheritance.
type Strategy struct {
	*crawlerdetect.UniversalStrategy
}

// InitStrategy is a function that creates and initializes an instance of the
// Strategy struct. It sets up the embedded UniversalStrategy with a pre-defined
// list of known crawler hostnames of Baidu, a popular search engine in China and Japan.
//
// Returns:
// - *Strategy: A pointer to an instance of the Strategy struct. Thanks to the
// predefined list of hosts in the `UniversalStrategy`, this `Strategy`
// instance is ready to detect crawlers from Baidu.
func InitStrategy() *Strategy {
	return &Strategy{
		UniversalStrategy: crawlerdetect.InitUniversalStrategy([]string{"baidu.com", "baidu.jp"}),
	}
}
