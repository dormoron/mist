package crawlerdetect

// BaiduStrategy is a struct used for detecting web crawlers. It embeds a pointer
// to an instance of the UniversalStrategy from the crawlerdetect package.
// By embedding it, the BaiduStrategy struct can directly call the methods
// and use the properties of the `UniversalStrategy`, achieving behavior akin to inheritance.
type BaiduStrategy struct {
	*UniversalStrategy
}

// InitBaiduStrategy is a function that creates and initializes an instance of the
// BaiduStrategy struct. It sets up the embedded UniversalStrategy with a pre-defined
// list of known crawler hostnames of Baidu, a popular search engine in China and Japan.
//
// Returns:
// - *BaiduStrategy: A pointer to an instance of the BaiduStrategy struct. Thanks to the
// predefined list of hosts in the `UniversalStrategy`, this `BaiduStrategy`
// instance is ready to detect crawlers from Baidu.
func InitBaiduStrategy() *BaiduStrategy {
	return &BaiduStrategy{
		UniversalStrategy: InitUniversalStrategy([]string{"baidu.com", "baidu.jp"}),
	}
}
