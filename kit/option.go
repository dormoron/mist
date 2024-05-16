package kit

// Option is a function type that applies a configuration to a type parameter 'T'.
// Parameters:
// - t: A pointer to an instance of a generic type 'T' which the option will configure.
type Option[T any] func(t *T) // No return value, modifies 'T' in-place.

// Apply sequentially applies a variadic slice of Option functions to an instance of type 'T'.
// This allows setting or modifying configurations of 'T' in a functional style.
// Parameters:
// - t: A pointer to the instance of type 'T' to be configured.
// - opts: A variadic slice of Option functions that will be applied to 'T'.
func Apply[T any](t *T, opts ...Option[T]) {
	for _, opt := range opts { // Iterate through each option function provided.
		opt(t) // Apply the option function to the instance 't'.
	}
	// There is no return value; the instance 't' is modified directly by the Option functions.
}

// OptionErr is like Option but expects an error return.
// This allows error checking after configuring the given instance of 'T'.
// Parameters:
// - t: A pointer to an instance of a generic type 'T' which the option will configure.
type OptionErr[T any] func(t *T) error // Returns an error which is nil if the application is successful.

// ApplyErr sequentially applies a variadic slice of OptionErr functions to an instance of type 'T'.
// If any OptionErr function returns an error, the process halts and the error is returned.
// Parameters:
// - t:    A pointer to the instance of type 'T' to be configured.
// - opts: A variadic slice of OptionErr functions that will be applied to 'T'.
// Returns:
// - error: The first error encountered during the application of OptionErr functions, or nil if all apply successfully.
func ApplyErr[T any](t *T, opts ...OptionErr[T]) error {
	for _, opt := range opts { // Iterate through each option function provided.
		if err := opt(t); err != nil {
			// If an OptionErr function returns an error, stop processing further
			// and return the error encountered.
			return err
		}
	}
	// Return nil indicating all OptionErr functions were applied successfully.
	return nil
}
