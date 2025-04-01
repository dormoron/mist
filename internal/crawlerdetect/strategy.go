package crawlerdetect

import (
	"net"
	"slices"
	"strings"
)

// Declaration of constants that represent four different search engines.
const (
	// Baidu is the largest search engine in China, providing various services beyond search such as maps, news, images, etc.
	Baidu = "baidu"

	// Bing is a web search engine owned and handled by Microsoft. It provides web search services to users and has a variety of features such as search, video, images, maps, and more.
	Bing = "bing"

	// Google is globally recognized and is the most used search engine. It handles over three billion searches each day and offers services beyond search like Gmail, Google Docs, etc.
	Google = "google"

	// SoGou is another Chinese search engine, owned by Sohu, Inc. It's the default search engine of Tencent's QQ soso.com, sogou.com, and Firefox in China.
	SoGou = "sogou"
)

// strategyMap is a mapping from search engine names to their respective crawler
// checking strategies. This map is used to dynamically select the appropriate
// strategy based on the search engine involved.
var strategyMap = map[string]Strategy{
	// Baidu's strategy is initialized and associated with the "Baidu" key in the map.
	Baidu: InitBaiduStrategy(),

	// Bing's strategy is initialized and associated with the "Bing" key in the map.
	Bing: InitBingStrategy(),

	// Google's strategy is initialized and associated with the "Google" key in the map.
	Google: InitGoogleStrategy(),

	// SoGou's strategy is initialized and associated with the "SoGou" key in the map.
	SoGou: InitSogouStrategy(),
}

// BaiduStrategy is an interface defining the methods that all crawler check strategies
// must implement. Different search engines may have their own implementation of
// the BaiduStrategy interface to accommodate their specific methods for detecting crawlers.
type Strategy interface {
	// CheckCrawler is a method that takes an IP address as input and returns a boolean
	// and an error. The boolean indicates whether the given IP address belongs to a crawler
	// or bot, and the error provides details if something went wrong during the check.
	// The implementation of this method should contain the logic for determining crawler activity.
	CheckCrawler(ip string) (bool, error)
}

// UniversalStrategy is a structure that holds information relevant to a generic
// approach for checking crawlers across different search engines.
type UniversalStrategy struct {
	Hosts []string // Hosts is a slice of strings that contains the hostnames or IP addresses used to identify crawlers.
	// This list can be used for cross-referencing against the IP addresses of incoming requests
	// to determine if any of them are known crawlers.
}

// InitUniversalStrategy is a function that initializes a UniversalStrategy instance. It takes a slice of hosts
// as input, which represent the hostnames or IP addresses used to identify crawlers.
// The function returns a pointer to a new UniversalStrategy instance that embeds this input data.
//
// Parameters:
//
//   - Hosts: This is a slice of strings that hold hostnames/IP addresses for crawler identification.
//
// The function constructs a UniversalStrategy struct and sets its internal "Hosts" field to the
// input slice of hostnames/IP addresses.
func InitUniversalStrategy(hosts []string) *UniversalStrategy {
	return &UniversalStrategy{
		Hosts: hosts,
	}
}

// CheckCrawler is a method associated with the UniversalStrategy struct, intended to
// determine if a given IP address belongs to a known crawler, typically employed by
// search engines. It operates by performing a reverse lookup of the IP to obtain hostnames
// and then matching these against the UniversalStrategy's list of hosts that are known to
// be crawlers.
//
// Parameters:
//
//   - IP: A string representing the IP address to be checked.
//
// Returns:
//
//   - bool: True if the IP address is identified as a crawler, false otherwise.
//   - error: Any error encountered during the execution of the IP lookup or subsequent operations.
//
// The method performs the following steps:
//
//  1. It executes a reverse DNS lookup on the given IP address to retrieve associated hostnames.
//  2. If an error occurs during the lookup, it returns false along with the error.
//  3. If no hostnames are found, it means the IP cannot be linked to any crawler and returns false.
//  4. If hostnames are found, it attempts to match them with known crawler hosts in the UniversalStrategy's list.
//     This is done through a custom matchHost method that is not shown here.
//  5. If there's no match, it returns false.
//  6. If a match is found, it then performs a forward IP lookup for the matched hostname to verify the IP address.
//  7. If the forward lookup yields an error, it returns false and the error.
//  8. Finally, it checks if the list of IPs from the forward lookup of the hostname contains the original IP address.
//     If so, it confirms the IP address belongs to a known crawler and returns true; otherwise, it returns false.
func (s *UniversalStrategy) CheckCrawler(ip string) (bool, error) {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return false, err
	}
	if len(names) == 0 {
		return false, nil
	}

	name, matched := s.matchHost(names)
	if !matched {
		return false, nil
	}

	ips, err := net.LookupIP(name)
	if err != nil {
		return false, err
	}
	// The slices package is used to find if any IP from the lookup matches the input IP.
	// Note: The slices package and ContainsFunc may require Go 1.18 or above.
	if slices.ContainsFunc(ips, func(netIp net.IP) bool {
		return netIp.String() == ip
	}) {
		return true, nil
	}

	return false, nil
}

// matchHost is a method of the UniversalStrategy struct that checks if any of the provided names
// match the hosts known to the strategy, typically used to identify crawlers. It aims to determine
// if any of the hostnames obtained from an IP address lookup match the predefined list of crawler hosts
// stored in the UniversalStrategy. This method supports identifying whether a particular IP belongs to
// a crawler based on DNS reverse lookup results.
//
// Parameters:
//
//   - names: A slice of strings containing hostnames retrieved from a reverse DNS lookup.
//
// Returns:
//
//   - string: The name of the matched host if found, otherwise an empty string.
//   - bool: True if a match is found, False otherwise.
//
// The method operates as follows:
//
//  1. Initialize an empty string `matchedName` to temporarily store the name
//     of a host if a match is found.
//  2. Utilize the slices.ContainsFunc method to iterate over each host string in the UniversalStrategy's
//     Hosts slice. For each host, the inner slices.ContainsFunc iterates over each provided `name`.
//  3. Within the nested checks, use strings.Contains to assess if the current `name` from the
//     reverse DNS lookup contains the current host being checked. This is a simple form of substring
//     matching.
//  4. If a match is found (i.e., `name` includes `host`), set `matchedName` to the current `name` to
//     capture the matched hostname and return true from the innermost lambda, indicating a match is found.
//  5. If the outer slices.ContainsFunc detects a match through the inner logic, it immediately returns
//     true, concluding the search with a match.
//  6. Finally, return `matchedName` and the result of the match check.
//
// Note: This function leverages the `slices` package for the ContainsFunc method, which could require
// Go version 1.18 or above. Additionally, it assumes that a partial match between the host string from
// UniversalStrategy's Hosts and any of the DNS names is sufficient to identify a crawler.
func (s *UniversalStrategy) matchHost(names []string) (string, bool) {
	var matchedName string
	return matchedName, slices.ContainsFunc(s.Hosts, func(host string) bool {
		return slices.ContainsFunc(names, func(name string) bool {
			if strings.Contains(name, host) {
				matchedName = name
				return true
			}
			return false
		})
	})
}

// InitCrawlerDetector is a function that retrieves a BaiduStrategy instance from a pre-defined
// map of strategies called strategyMap, based on the specified crawler string.
// Each crawler in the map is associated with a specific initialization function for its strategy,
// which is assumed to have been initialized earlier and stored in the strategyMap.
// This function acts as a lookup to fetch the appropriate BaiduStrategy instance for a given crawler.
//
// Parameters:
//
//   - crawler: A string that identifies the crawler whose BaiduStrategy instance needs to be retrieved.
//     It acts as a key to the strategyMap.
//
// Returns:
//
//   - BaiduStrategy: A BaiduStrategy instance associated with the provided crawler string. If the crawler string
//     does not exist in the map, the function returns a nil value.
//
// Usage Notes:
//
//   - The strategyMap is a global variable where the key is a string representing the crawler's name,
//     and the value is an instance of a BaiduStrategy implementation specific to that crawler.
//   - The provided crawler string should match one of the keys in the strategyMap for the function
//     to return a valid BaiduStrategy instance.
//   - If the crawler string is not found in the strategyMap or is misspelled, the function will return
//     a nil value, which the caller must check for before proceeding to use the returned BaiduStrategy instance.
//
// Example:
//
//   - To retrieve the BaiduStrategy associated with 'Google', you would call:
//     googleStrategy := InitCrawlerDetector("Google")
func InitCrawlerDetector(crawler string) Strategy {
	return strategyMap[crawler]
}
