package bloom

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"time"
)

// RedisBloomFilter represents a Bloom filter backed by Redis.
type RedisBloomFilter struct {
	client  redis.Cmdable // Redis client to execute commands.
	options *Options      // Options for the Bloom filter such as Redis key, etc.
	mu      sync.Mutex    // Mutex to ensure thread-safe operations.
}

// InitRedisBloomFilter initializes a new Bloom filter with given Redis client and optional settings.
func InitRedisBloomFilter(client redis.Cmdable, opts ...Option) Filter {
	// Initialize default options.
	options := defaultOptions()

	// Apply optional settings.
	for _, opt := range opts {
		opt(options)
	}

	// Return a new instance of RedisBloomFilter.
	return &RedisBloomFilter{
		client:  client,
		options: options,
	}
}

// Add inserts multiple elements into the Bloom filter.
// Parameters:
// - ctx: Context to control execution.
// - elements: Variadic parameter for the elements to add.
func (bf *RedisBloomFilter) Add(ctx context.Context, elements ...interface{}) error {
	// Ensure that only one Add operation can run at a time.
	bf.mu.Lock()
	defer bf.mu.Unlock()

	// Check if there are elements to add.
	if len(elements) == 0 {
		return errors.New("no elements to add")
	}

	// Convert all elements to string format.
	args := make([]interface{}, len(elements))
	for i, elem := range elements {
		args[i] = fmt.Sprintf("%v", elem)
	}

	// Execute Lua script to add elements to the Bloom filter.
	_, err := bf.client.Eval(ctx, addLuaScript, []string{bf.options.RedisKey}, args...).Result()
	if err != nil {
		log.Printf("Error adding elements to BloomFilter: %v", err)

		// Retry on error.
		return retryOnError(ctx, func() error {
			_, errRetry := bf.client.Eval(ctx, addLuaScript, []string{bf.options.RedisKey}, args...).Result()
			if errRetry != nil {
				log.Printf("Retry error adding elements to BloomFilter: %v", errRetry)
			}
			return errRetry
		})
	}

	return err
}

// Check checks if a single element exists in the Bloom filter.
// Parameters:
// - ctx: Context to control execution.
// - element: The element to check.
// Returns:
// - bool: true if the element exists, false otherwise.
// - error: any error encountered during the operation.
func (bf *RedisBloomFilter) Check(ctx context.Context, element interface{}) (bool, error) {
	// Use batch check internally for consistency.
	exists, err := bf.CheckBatch(ctx, element)
	if err != nil {
		return false, err
	}
	if len(exists) > 0 {
		return exists[0], nil
	}
	return false, nil
}

// CheckBatch checks if multiple elements exist in the Bloom filter.
// Parameters:
// - ctx: Context to control execution.
// - elements: Variadic parameter for the elements to check.
// Returns:
// - []bool: Slice of boolean values indicating existence of each element.
// - error: any error encountered during the operation.
func (bf *RedisBloomFilter) CheckBatch(ctx context.Context, elements ...interface{}) ([]bool, error) {
	// Ensure that only one batch check can run at a time.
	bf.mu.Lock()
	defer bf.mu.Unlock()

	// Convert all elements to string format.
	args := make([]interface{}, len(elements))
	for i, elem := range elements {
		args[i] = fmt.Sprintf("%v", elem)
	}

	// Execute Lua script to check elements in the Bloom filter.
	results, err := bf.client.Eval(ctx, checkLuaScript, []string{bf.options.RedisKey}, args...).Result()
	if err != nil {
		log.Printf("Error checking elements in BloomFilter: %v", err)

		// Retry on error.
		return nil, retryOnError(ctx, func() error {
			resultsRetry, errRetry := bf.client.Eval(ctx, checkLuaScript, []string{bf.options.RedisKey}, args...).Result()
			if errRetry == nil {
				results = resultsRetry
			}
			return errRetry
		})
	}

	// Convert results to boolean slice.
	boolResults := make([]bool, len(elements))
	for i, res := range results.([]interface{}) {
		boolResults[i] = res.(int64) == 1
	}

	return boolResults, err
}

// Remove removes a single element from the Bloom filter.
// This is a convenience method that calls RemoveBatch internally.
// Parameters:
// - ctx: Context to control execution.
// - element: The element to remove.
func (bf *RedisBloomFilter) Remove(ctx context.Context, element interface{}) error {
	return bf.RemoveBatch(ctx, element)
}

// RemoveBatch removes multiple elements from the Bloom filter.
// Parameters:
// - ctx: Context to control execution.
// - elements: Variadic parameter for the elements to remove.
// Returns:
// - error: any error encountered during the operation.
func (bf *RedisBloomFilter) RemoveBatch(ctx context.Context, elements ...interface{}) error {
	// Ensure that only one batch remove can run at a time.
	bf.mu.Lock()
	defer bf.mu.Unlock()

	// Convert all elements to string format.
	args := make([]interface{}, len(elements))
	for i, elem := range elements {
		args[i] = fmt.Sprintf("%v", elem)
	}

	// Execute Lua script to remove elements from the Bloom filter.
	_, err := bf.client.Eval(ctx, removeCountLuaScript, []string{bf.options.RedisKey}, args...).Result()
	if err != nil {
		log.Printf("Error removing elements from BloomFilter: %v", err)

		// Retry on error.
		return retryOnError(ctx, func() error {
			_, errRetry := bf.client.Eval(ctx, removeCountLuaScript, []string{bf.options.RedisKey}, args...).Result()
			if errRetry != nil {
				log.Printf("Retry error removing elements from BloomFilter: %v", errRetry)
			}
			return errRetry
		})
	}

	return err
}

// retryOnError retries a function execution on failure with exponential backoff.
// Parameters:
// - ctx: Context to control execution.
// - fn: Function to retry.
// Returns:
// - error: the last error encountered after retries, or nil if successful.
func retryOnError(ctx context.Context, fn func() error) error {
	// Define retry parameters.
	retries := 3
	backoff := time.Millisecond * 100

	// Try the function up to the number of retries.
	for i := 0; i < retries; i++ {
		if err := fn(); err != nil {
			if i < retries-1 {
				// Sleep and increment backoff for next retry.
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return err
		}
		// If successful, return nil.
		return nil
	}
	return nil
}
