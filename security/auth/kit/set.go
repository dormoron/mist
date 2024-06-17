package kit

// Set is a generic interface representing a set data structure. It provides
// methods to add, delete, check the existence of elements, and retrieve all elements as a slice.
type Set[T comparable] interface {
	Add(key T)        // Adds a new element to the set.
	Delete(key T)     // Deletes an element from the set.
	Exist(key T) bool // Checks if an element exists in the set.
	Keys() []T        // Retrieves all elements in the set as a slice.
}

// MapSet is a generic implementation of the Set interface using a Go map.
// It ensures that elements are unique within the set.
type MapSet[T comparable] struct {
	m map[T]struct{} // Underlying map to store set elements. The value is an empty struct to save memory.
}

// NewMapSet creates and returns a new instance of MapSet with an initial capacity.
// Parameters:
// - size: The initial capacity of the map (int).
// Returns:
// - *MapSet[T]: A pointer to a newly created MapSet.
func NewMapSet[T comparable](size int) *MapSet[T] {
	return &MapSet[T]{
		m: make(map[T]struct{}, size), // Initialize the map with the given capacity.
	}
}

// Add inserts a new element into the MapSet. If the element already exists, it does nothing.
// Parameters:
// - val: The value to be added to the set (T).
func (s *MapSet[T]) Add(val T) {
	s.m[val] = struct{}{} // Add the value to the map with an empty struct value.
}

// Delete removes an element from the MapSet. If the element does not exist, it does nothing.
// Parameters:
// - key: The value to be removed from the set (T).
func (s *MapSet[T]) Delete(key T) {
	delete(s.m, key) // Delete the value from the map.
}

// Exist checks if a given element exists in the MapSet.
// Parameters:
// - key: The value to check for existence in the set (T).
// Returns:
// - bool: True if the value exists in the set, false otherwise.
func (s *MapSet[T]) Exist(key T) bool {
	_, ok := s.m[key]
	return ok // Return whether the key exists in the map.
}

// Keys retrieves all elements from the MapSet as a slice.
// Returns:
// - []T: A slice containing all elements in the set.
func (s *MapSet[T]) Keys() []T {
	ans := make([]T, 0, len(s.m)) // Initialize a slice with the length of the map.
	for key := range s.m {
		ans = append(ans, key) // Append each key from the map to the slice.
	}
	return ans
}
