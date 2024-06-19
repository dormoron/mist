package bloom

// Options holds configuration settings for the RedisBloomFilter.
type Options struct {
	RedisKey string // RedisKey is the key under which the Bloom filter data is stored in Redis.
}

// Option is a function type that takes an Options pointer and configures it.
type Option func(*Options)

// defaultOptions returns a pointer to an Options struct with default values.
// Returns:
// - *Options: A pointer to the Options struct with RedisKey set to a default value.
func defaultOptions() *Options {
	return &Options{
		RedisKey: "bloom_filter", // Default key used to store Bloom filter in Redis.
	}
}

// WithRedisKey returns an Option that sets the RedisKey in Options.
// Parameters:
// - key: The Redis key to be set in Options.
// Returns:
// - Option: A function that sets the RedisKey in Options.
func WithRedisKey(key string) Option {
	return func(opts *Options) {
		opts.RedisKey = key // Configures the RedisKey value in the Options struct.
	}
}
