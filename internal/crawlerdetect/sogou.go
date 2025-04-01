package crawlerdetect

import (
	"net"
	"slices"
	"strings"
)

// SogouStrategy is a struct that holds information needed to check if an IP address
// is associated with a known web crawler. This is often used to differentiate between
// regular user traffic and automated crawlers, such as those used by search engines.
//
// Fields:
//   - Hosts: A slice of strings where each string is a host that is known to
//     be associated with a web crawler. For instance, "googlebot.com" for Google's crawler.
type SogouStrategy struct {
	Hosts []string
}

// InitSogouStrategy is a package-level function that initializes a SogouStrategy struct
// with predefined host names of known crawlers. This example uses "sogou.com" as a
// known crawler host.
//
// Returns:
// - *SogouStrategy: A pointer to a SogouStrategy instance with prepopulated Hosts field.
func InitSogouStrategy() *SogouStrategy {
	return &SogouStrategy{
		Hosts: []string{"sogou.com"},
	}
}

// CheckCrawler is a method linked to the SogouStrategy struct that attempts to
// verify if a given IP address belongs to a known web crawler defined in the struct's Hosts field.
//
// Parameters:
// - ip: The IP address to check against the list of known crawler hosts.
//
// Returns:
// - bool: Indicates whether the IP is a known crawler (`true`) or not (`false`).
// - error: Any error encountered during the DNS look-up process.
//
// The method performs a reverse DNS lookup of the IP address to ascertain if any associated
// hosts match the ones listed in the SogouStrategy's Hosts field using the matchHost method.
func (s *SogouStrategy) CheckCrawler(ip string) (bool, error) {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return false, err
	}
	if len(names) == 0 {
		return false, nil
	}
	return s.matchHost(names), nil
}

// matchHost is a helper method for the SogouStrategy struct that checks if any of the hostnames
// returned from a reverse DNS lookup match the hosts known to be crawlers.
//
// Parameters:
// - names: A slice of hostnames obtained from the reverse DNS lookup of an IP address.
//
// Returns:
// - bool: Whether any of the provided names match the known crawler hosts.
//
// It uses the slices.ContainsFunc method to iterate over the list of known hosts and compares
// each with the retrieved names using the strings.Contains method. If a match is found
// the function returns true, otherwise it returns false.
func (s *SogouStrategy) matchHost(names []string) bool {
	return slices.ContainsFunc(s.Hosts, func(host string) bool {
		return slices.ContainsFunc(names, func(name string) bool {
			return strings.Contains(name, host)
		})
	})
}
