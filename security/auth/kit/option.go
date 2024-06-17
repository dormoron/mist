package kit

// Option is a generic type for a functional option, which is a function
// that configures a given instance of type T.
type Option[T any] func(t *T)

// Apply applies a variadic list of functional options to a given instance of type T.
// Parameters:
// - t: A pointer to the instance to be configured (*T).
// - opts: A variadic list of functional options (Option[T]).
// The function iterates over the provided options and applies each one to the instance.
func Apply[T any](t *T, opts ...Option[T]) {
	for _, opt := range opts {
		opt(t) // Apply each option to the instance.
	}
}

// OptionErr is a generic type for a functional option, which is a function
// that configures a given instance of type T and may return an error.
type OptionErr[T any] func(t *T) error

// ApplyErr applies a variadic list of functional options to a given instance of type T,
// and returns an error if any option fails.
// Parameters:
// - t: A pointer to the instance to be configured (*T).
// - opts: A variadic list of functional options that may return errors (OptionErr[T]).
// Returns:
// - error: An error if any of the options return an error, otherwise nil.
// The function iterates over the provided options and applies each one to the instance.
// If any option returns an error, it is immediately returned and no further options are applied.
func ApplyErr[T any](t *T, opts ...OptionErr[T]) error {
	for _, opt := range opts {
		if err := opt(t); err != nil { // Apply each option to the instance and check for errors.
			return err // Return the error if any option fails.
		}
	}
	return nil // Return nil if no errors occur.
}
