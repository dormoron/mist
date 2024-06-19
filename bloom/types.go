package bloom

import (
	"context"
	_ "embed"
)

var (
	// `addLuaScript` contains the Lua script for adding elements to the Bloom filter.
	// This script is embedded from the `lua/add.lua` file at compile time.
	//go:embed lua/add.lua
	addLuaScript string

	// `checkLuaScript` contains the Lua script for checking elements in the Bloom filter.
	// This script is embedded from the `lua/check.lua` file at compile time.
	//go:embed lua/check.lua
	checkLuaScript string

	// `removeCountLuaScript` contains the Lua script for removing elements from the Bloom filter.
	// This script is embedded from the `lua/remove.lua` file at compile time.
	//go:embed lua/remove.lua
	removeCountLuaScript string
)

// Filter is an interface that defines methods for interacting with a Bloom filter.
type Filter interface {
	// Add inserts multiple elements into the Bloom filter.
	// Parameters:
	// - ctx: Context to control the execution and allow cancellation.
	// - elements: Variadic parameter for the elements to add to the Bloom filter.
	// Returns:
	// - error: An error if the addition fails, otherwise nil.
	Add(ctx context.Context, elements ...interface{}) error

	// Check verifies whether a single element is present in the Bloom filter.
	// Parameters:
	// - ctx: Context to control the execution and allow cancellation.
	// - element: The element to check in the Bloom filter.
	// Returns:
	// - bool: true if the element may be present, false otherwise.
	// - error: An error if the check operation fails, otherwise nil.
	Check(ctx context.Context, element interface{}) (bool, error)

	// CheckBatch verifies the presence of multiple elements in the Bloom filter.
	// Parameters:
	// - ctx: Context to control the execution and allow cancellation.
	// - elements: Variadic parameter for the elements to check in the Bloom filter.
	// Returns:
	// - []bool: Slice of boolean values indicating the presence of each element.
	// - error: An error if the check operation fails, otherwise nil.
	CheckBatch(ctx context.Context, elements ...interface{}) ([]bool, error)

	// Remove deletes a single element from the Bloom filter.
	// Parameters:
	// - ctx: Context to control the execution and allow cancellation.
	// - element: The element to remove from the Bloom filter.
	// Returns:
	// - error: An error if the removal fails, otherwise nil.
	Remove(ctx context.Context, element interface{}) error

	// RemoveBatch deletes multiple elements from the Bloom filter.
	// Parameters:
	// - ctx: Context to control the execution and allow cancellation.
	// - elements: Variadic parameter for the elements to remove from the Bloom filter.
	// Returns:
	// - error: An error if the removal fails, otherwise nil.
	RemoveBatch(ctx context.Context, elements ...interface{}) error
}
