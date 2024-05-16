package kit

import (
	"fmt"
)

// AnyValue wraps a value of any type along with an error.
// It's useful for functions that return a value and an error to be able to package both into a single return value.
type AnyValue struct {
	Val any   // Val is a value of any type.
	Err error // Err contains an error if it occurred during the value's processing.
}

// String tries to convert AnyValue's Val to a string.
// If Val is not a string or if an error is stored in Err, it returns an error.
// returns the string representation of Val, and an error if the conversion is not possible or an error was previously recorded in Err.
func (av AnyValue) String() (string, error) {
	if av.Err != nil {
		// If AnyValue contains an error, return an empty string and the error.
		return "", av.Err
	}
	val, ok := av.Val.(string) // Attempt to assert Val's type to string.
	if !ok {
		// If the type assertion fails, return an error indicating the expected type and the actual value.
		return "", fmt.Errorf("type conversion failed, expected type:%s, actual value:%#v", "string", av.Val)
	}
	return val, nil // Return the string and a nil error on success.
}

// StringOrDefault attempts to convert AnyValue's Val to a string, returning a default value if the conversion is unsuccessful or an error exists.
// def: The default string value to return in case of an error or conversion failure.
// returns the string representation of Val, or the default value 'def' if an error occurs or the conversion is not possible.
func (av AnyValue) StringOrDefault(def string) string {
	val, err := av.String() // Attempt to convert Val to a string.
	if err != nil {
		// If String() returns an error, return the default value.
		return def
	}
	return val // Return the converted string value if no error occurred.
}
