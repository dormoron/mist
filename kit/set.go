package kit

// Set is an interface for a collection of comparable elements with no duplicate values.
// T: Type parameter which must be comparable.
type Set[T comparable] interface {
	// Add includes an element in the set.
	Add(key T)
	// Delete removes an element from the set.
	Delete(key T)
	// Exist checks if the set contains an element.
	Exist(key T) bool
	// Keys returns all elements in the set.
	Keys() []T
}

// MapSet is a concrete implementation of Set interface, using a map data structure.
// T: Type parameter which must be comparable.
type MapSet[T comparable] struct {
	// The map where the key represents the element in the set. We use an empty struct as the value because it doesn't occupy additional memory.
	m map[T]struct{}
}

// InitMapSet initializes a MapSet instance with a specified size.
// T: Type parameter which must be comparable.
// size: The initial size of the MapSet.
// returns a pointer to a MapSet instance.
func InitMapSet[T comparable](size int) *MapSet[T] {
	return &MapSet[T]{
		m: make(map[T]struct{}, size), // Initialize the underlying map with the specified size for performance reasons.
	}
}

// Add includes a new element in the MapSet.
// val: The value to be added to the MapSet.
func (s *MapSet[T]) Add(val T) {
	s.m[val] = struct{}{} // Store the element in the map. Set the value to an empty struct because we are only interested in the keys (elements).
}

// Delete removes an element from the MapSet.
// key: The element to be removed from the MapSet.
func (s *MapSet[T]) Delete(key T) {
	delete(s.m, key) // Removes the key-value pair from the map.
}

// Exist checks if the MapSet contains a specific element.
// key: The element to be checked in the MapSet.
// returns a boolean indicating whether the element exists in the MapSet.
func (s *MapSet[T]) Exist(key T) bool {
	_, ok := s.m[key] // If key is present in the map, ok is true. Else, it is false.
	return ok
}

// Keys gathers all elements from the MapSet.
// returns a slice of all elements in the MapSet.
func (s *MapSet[T]) Keys() []T {
	ans := make([]T, 0, len(s.m)) // Prepare a slice to store the results with the correct capacity.
	for key := range s.m {
		ans = append(ans, key) // Read all keys (elements) from the map and append it to the slice.
	}
	return ans
}
