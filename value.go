package mist

import (
	"encoding/json"
	"errors"
	"github.com/dormoron/mist/internal/errs"
	"reflect"
	"strconv"
)

// AnyValue provides a structure to store any value along with an optional error
type AnyValue struct {
	Val any   // Val represents the value of any type that this struct holds
	Err error // Err represents an optional error associated with the value
}

// Int attempts to convert the stored value to an int and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to int fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int: The integer value stored in the struct
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil
func (av AnyValue) Int() (int, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(int)
	if !ok {
		return 0, errs.ErrInvalidType("int", av.Val)
	}
	return val, nil
}

// AsInt tries to interpret the stored value as an int, handling string conversion to int if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses type switch to differentiate between int and string types.
// If the type is string, it tries to parse the string to an int.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int: The integer value after conversion
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil
func (av AnyValue) AsInt() (int, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case int:
		return v, nil
	case string:
		res, err := strconv.ParseInt(v, 10, 64)
		return int(res), err
	}
	return 0, errs.ErrInvalidType("int", av.Val)
}

// IntOrDefault returns the stored value as an int, or a default value if there is an error.
// It reuses the Int method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def int: The default integer value to return if there is an error
// Returns:
// - int: The integer value stored in the struct or the default value if an error occurs
func (av AnyValue) IntOrDefault(def int) int {
	val, err := av.Int()
	if err != nil {
		return def
	}
	return val
}

// Uint attempts to convert the stored value to a uint and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to uint fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint: The unsigned integer value stored in the struct
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil
func (av AnyValue) Uint() (uint, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(uint)
	if !ok {
		return 0, errs.ErrInvalidType("uint", av.Val)
	}
	return val, nil
}

// AsUint tries to interpret the stored value as a uint, handling string conversion to uint if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses type switch to differentiate between uint and string types.
// If the type is string, it tries to parse the string to a uint.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint: The unsigned integer value after conversion
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil
func (av AnyValue) AsUint() (uint, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case uint:
		return v, nil
	case string:
		res, err := strconv.ParseUint(v, 10, 64)
		return uint(res), err
	}
	return 0, errs.ErrInvalidType("uint", av.Val)
}

// UintOrDefault returns the stored value as a uint, or a default value if there is an error.
// It reuses the Uint method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def uint: The default unsigned integer value to return if there is an error
// Returns:
// - uint: The unsigned integer value stored in the struct or the default value if an error occurs
func (av AnyValue) UintOrDefault(def uint) uint {
	val, err := av.Uint()
	if err != nil {
		return def
	}
	return val
}

// Int8 attempts to convert the stored value to an int8 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to int8 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int8: The int8 value stored in the struct
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil
func (av AnyValue) Int8() (int8, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(int8)
	if !ok {
		return 0, errs.ErrInvalidType("int8", av.Val)
	}
	return val, nil
}

// AsInt8 tries to interpret the stored value as an int8, handling string conversion to int8 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses type switch to differentiate between int8 and string types.
// If the type is string, it tries to parse the string to an int8.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int8: The int8 value after conversion
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil
func (av AnyValue) AsInt8() (int8, error) {
	if av.Err != nil {
		return 0, av.Err
	}

	switch v := av.Val.(type) {
	case int8:
		return v, nil
	case string:
		res, err := strconv.ParseInt(v, 10, 64)
		return int8(res), err
	}
	return 0, errs.ErrInvalidType("int8", av.Val)
}

// Int8OrDefault returns the stored value as an int8, or a default value if there is an error.
// It reuses the Int8 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def int8: The default int8 value to return if there is an error
// Returns:
// - int8: The int8 value stored in the struct or the default value if an error occurs
func (av AnyValue) Int8OrDefault(def int8) int8 {
	val, err := av.Int8()
	if err != nil {
		return def
	}
	return val
}

// Uint8 attempts to convert the stored value to a uint8 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to uint8 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint8: The uint8 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Uint8() (uint8, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(uint8)
	if !ok {
		return 0, errs.ErrInvalidType("uint8", av.Val)
	}
	return val, nil
}

// AsUint8 tries to interpret the stored value as a uint8, handling string conversion to uint8 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between uint8 and string types.
// If the type is string, it tries to parse the string to a uint8.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint8: The uint8 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsUint8() (uint8, error) {
	if av.Err != nil {
		return 0, av.Err
	}

	switch v := av.Val.(type) {
	case uint8:
		return v, nil
	case string:
		res, err := strconv.ParseUint(v, 10, 8)
		return uint8(res), err
	}
	return 0, errs.ErrInvalidType("uint8", av.Val)
}

// Uint8OrDefault returns the stored value as a uint8, or a default value if there is an error.
// It reuses the Uint8 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def uint8: The default uint8 value to return if there is an error.
// Returns:
// - uint8: The uint8 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Uint8OrDefault(def uint8) uint8 {
	val, err := av.Uint8()
	if err != nil {
		return def
	}
	return val
}

// Int16 attempts to convert the stored value to an int16 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to int16 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int16: The int16 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Int16() (int16, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(int16)
	if !ok {
		return 0, errs.ErrInvalidType("int16", av.Val)
	}
	return val, nil
}

// AsInt16 tries to interpret the stored value as an int16, handling string conversion to int16 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between int16 and string types.
// If the type is string, it tries to parse the string to an int16.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int16: The int16 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsInt16() (int16, error) {
	if av.Err != nil {
		return 0, av.Err
	}

	switch v := av.Val.(type) {
	case int16:
		return v, nil
	case string:
		res, err := strconv.ParseInt(v, 10, 16)
		return int16(res), err
	}
	return 0, errs.ErrInvalidType("int16", av.Val)
}

// Int16OrDefault returns the stored value as an int16, or a default value if there is an error.
// It reuses the Int16 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def int16: The default int16 value to return if there is an error.
// Returns:
// - int16: The int16 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Int16OrDefault(def int16) int16 {
	val, err := av.Int16()
	if err != nil {
		return def
	}
	return val
}

// Uint16 attempts to convert the stored value to a uint16 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to uint16 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint16: The uint16 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Uint16() (uint16, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(uint16)
	if !ok {
		return 0, errs.ErrInvalidType("uint16", av.Val)
	}
	return val, nil
}

// AsUint16 tries to interpret the stored value as a uint16, handling string conversion to uint16 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between uint16 and string types.
// If the type is string, it tries to parse the string to a uint16.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint16: The uint16 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsUint16() (uint16, error) {
	if av.Err != nil {
		return 0, av.Err
	}

	switch v := av.Val.(type) {
	case uint16:
		return v, nil
	case string:
		res, err := strconv.ParseUint(v, 10, 16)
		return uint16(res), err
	}
	return 0, errs.ErrInvalidType("uint16", av.Val)
}

// Uint16OrDefault returns the stored value as a uint16, or a default value if there is an error.
// It reuses the Uint16 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def uint16: The default uint16 value to return if there is an error.
// Returns:
// - uint16: The uint16 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Uint16OrDefault(def uint16) uint16 {
	val, err := av.Uint16()
	if err != nil {
		return def
	}
	return val
}

// Int32 attempts to convert the stored value to an int32 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to int32 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int32: The int32 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Int32() (int32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(int32)
	if !ok {
		return 0, errs.ErrInvalidType("int32", av.Val)
	}
	return val, nil
}

// AsInt32 tries to interpret the stored value as an int32, handling string conversion to int32 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between int32 and string types.
// If the type is string, it tries to parse the string to an int32.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int32: The int32 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsInt32() (int32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case int32:
		return v, nil
	case string:
		res, err := strconv.ParseInt(v, 10, 32)
		return int32(res), err
	}
	return 0, errs.ErrInvalidType("int32", av.Val)
}

// Int32OrDefault returns the stored value as an int32, or a default value if there is an error.
// It reuses the Int32 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def int32: The default int32 value to return if there is an error.
// Returns:
// - int32: The int32 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Int32OrDefault(def int32) int32 {
	val, err := av.Int32()
	if err != nil {
		return def
	}
	return val
}

// Uint32 attempts to convert the stored value to a uint32 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to uint32 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint32: The uint32 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Uint32() (uint32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(uint32)
	if !ok {
		return 0, errs.ErrInvalidType("uint32", av.Val)
	}
	return val, nil
}

// AsUint32 tries to interpret the stored value as a uint32, handling string conversion to uint32 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between uint32 and string types.
// If the type is string, it tries to parse the string to a uint32.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint32: The uint32 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsUint32() (uint32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case uint32:
		return v, nil
	case string:
		res, err := strconv.ParseUint(v, 10, 32)
		return uint32(res), err
	}
	return 0, errs.ErrInvalidType("uint32", av.Val)
}

// Uint32OrDefault returns the stored value as a uint32, or a default value if there is an error.
// It reuses the Uint32 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def uint32: The default uint32 value to return if there is an error.
// Returns:
// - uint32: The uint32 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Uint32OrDefault(def uint32) uint32 {
	val, err := av.Uint32()
	if err != nil {
		return def
	}
	return val
}

// Int64 attempts to convert the stored value to an int64 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to int64 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int64: The int64 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Int64() (int64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(int64)
	if !ok {
		return 0, errs.ErrInvalidType("int64", av.Val)
	}
	return val, nil
}

// AsInt64 tries to interpret the stored value as an int64, handling string conversion to int64 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between int64 and string types.
// If the type is string, it tries to parse the string to an int64.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - int64: The int64 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsInt64() (int64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, errs.ErrInvalidType("int64", av.Val)
}

// Int64OrDefault returns the stored value as an int64, or a default value if there is an error.
// It reuses the Int64 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def int64: The default int64 value to return if there is an error.
// Returns:
// - int64: The int64 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Int64OrDefault(def int64) int64 {
	val, err := av.Int64()
	if err != nil {
		return def
	}
	return val
}

// Uint64 attempts to convert the stored value to a uint64 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to uint64 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint64: The uint64 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Uint64() (uint64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(uint64)
	if !ok {
		return 0, errs.ErrInvalidType("uint64", av.Val)
	}
	return val, nil
}

// AsUint64 tries to interpret the stored value as a uint64, handling string conversion to uint64 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between uint64 and string types.
// If the type is string, it tries to parse the string to a uint64.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - uint64: The uint64 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsUint64() (uint64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case uint64:
		return v, nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	}
	return 0, errs.ErrInvalidType("uint64", av.Val)
}

// Uint64OrDefault returns the stored value as a uint64, or a default value if there is an error.
// It reuses the Uint64 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def uint64: The default uint64 value to return if there is an error.
// Returns:
// - uint64: The uint64 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Uint64OrDefault(def uint64) uint64 {
	val, err := av.Uint64()
	if err != nil {
		return def
	}
	return val
}

// Float32 attempts to convert the stored value to a float32 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to float32 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - float32: The float32 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Float32() (float32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(float32)
	if !ok {
		return 0, errs.ErrInvalidType("float32", av.Val)
	}
	return val, nil
}

// AsFloat32 tries to interpret the stored value as a float32, handling string conversion to float32 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between float32 and string types.
// If the type is string, it tries to parse the string to a float32 using strconv.ParseFloat.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - float32: The float32 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsFloat32() (float32, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case float32:
		return v, nil
	case string:
		res, err := strconv.ParseFloat(v, 32)
		return float32(res), err
	}
	return 0, errs.ErrInvalidType("float32", av.Val)
}

// Float32OrDefault returns the stored value as a float32, or a default value if there is an error.
// It reuses the Float32 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def float32: The default float32 value to return if there is an error.
// Returns:
// - float32: The float32 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Float32OrDefault(def float32) float32 {
	val, err := av.Float32()
	if err != nil {
		return def
	}
	return val
}

// Float64 attempts to convert the stored value to a float64 and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to float64 fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - float64: The float64 value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Float64() (float64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	val, ok := av.Val.(float64)
	if !ok {
		return 0, errs.ErrInvalidType("float64", av.Val)
	}
	return val, nil
}

// AsFloat64 tries to interpret the stored value as a float64, handling string conversion to float64 if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between float64 and string types.
// If the type is string, it tries to parse the string to a float64 using strconv.ParseFloat.
// If both assertions fail, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - float64: The float64 value after conversion.
// - error: An optional error if something went wrong during type assertion or parsing, or if av.Err is not nil.
func (av AnyValue) AsFloat64() (float64, error) {
	if av.Err != nil {
		return 0, av.Err
	}
	switch v := av.Val.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	}
	return 0, errs.ErrInvalidType("float64", av.Val)
}

// Float64OrDefault returns the stored value as a float64, or a default value if there is an error.
// It reuses the Float64 method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def float64: The default float64 value to return if there is an error.
// Returns:
// - float64: The float64 value stored in the struct or the default value if an error occurs.
func (av AnyValue) Float64OrDefault(def float64) float64 {
	val, err := av.Float64()
	if err != nil {
		return def
	}
	return val
}

// String attempts to convert the stored value to a string and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to string fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - string: The string value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) String() (string, error) {
	if av.Err != nil {
		return "", av.Err
	}
	val, ok := av.Val.(string)
	if !ok {
		return "", errs.ErrInvalidType("string", av.Val)
	}
	return val, nil
}

// AsString tries to convert various numeric and slice types to a string representation.
// If av.Err is not nil, the function returns the error immediately.
// It uses reflection to handle different types such as uint, int, float32, float64, and byte slices ([]byte).
//
// Parameters: None
// Returns:
// - string: The string value after conversion.
// - error: An optional error if something went wrong during type assertion or conversion, or if av.Err is not nil.
func (av AnyValue) AsString() (string, error) {
	if av.Err != nil {
		return "", av.Err
	}

	var val string
	valueOf := reflect.ValueOf(av.Val)
	switch valueOf.Type().Kind() {
	case reflect.String:
		val = valueOf.String()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val = strconv.FormatUint(valueOf.Uint(), 10)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = strconv.FormatInt(valueOf.Int(), 10)
	case reflect.Float32:
		val = strconv.FormatFloat(valueOf.Float(), 'f', 10, 32)
	case reflect.Float64:
		val = strconv.FormatFloat(valueOf.Float(), 'f', 10, 64)
	case reflect.Slice:
		if valueOf.Type().Elem().Kind() != reflect.Uint8 {
			return "", errs.ErrInvalidType("[]byte", av.Val)
		}
		val = string(valueOf.Bytes())
	default:
		return "", errors.New("unsupported type, conversion not possible")
	}

	return val, nil
}

// StringOrDefault returns the stored value as a string, or a default value if there is an error.
// It reuses the String method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def string: The default string value to return if there is an error.
// Returns:
// - string: The string value stored in the struct or the default value if an error occurs.
func (av AnyValue) StringOrDefault(def string) string {
	val, err := av.String()
	if err != nil {
		return def
	}
	return val
}

// Bytes attempts to convert the stored value to a byte slice ([]byte) and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to []byte fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - []byte: The byte slice value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Bytes() ([]byte, error) {
	if av.Err != nil {
		return nil, av.Err
	}
	val, ok := av.Val.([]byte)
	if !ok {
		return nil, errs.ErrInvalidType("[]byte", av.Val)
	}
	return val, nil
}

// AsBytes tries to interpret the stored value as a byte slice ([]byte), handling string conversion to []byte if necessary.
// If av.Err is not nil, the function returns the error immediately.
// It uses a type switch to differentiate between []byte and string types.
// If the type is string, it converts the string to a byte slice ([]byte).
//
// Parameters: None
// Returns:
// - []byte: The byte slice value after conversion.
// - error: An optional error if something went wrong during type assertion or conversion, or if av.Err is not nil.
func (av AnyValue) AsBytes() ([]byte, error) {
	if av.Err != nil {
		return []byte{}, av.Err
	}
	switch v := av.Val.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	}

	return []byte{}, errs.ErrInvalidType("[]byte", av.Val)
}

// BytesOrDefault returns the stored value as a byte slice ([]byte), or a default value if there is an error.
// It reuses the Bytes method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def []byte: The default byte slice to return if there is an error.
// Returns:
// - []byte: The byte slice value stored in the struct or the default value if an error occurs.
func (av AnyValue) BytesOrDefault(def []byte) []byte {
	val, err := av.Bytes()
	if err != nil {
		return def
	}
	return val
}

// Bool attempts to convert the stored value to a boolean and returns it, along with any error.
// If av.Err is not nil, the function returns the error immediately.
// If the type assertion to bool fails, it returns a custom error indicating the value type mismatch.
//
// Parameters: None
// Returns:
// - bool: The boolean value stored in the struct.
// - error: An optional error if something went wrong during type assertion or if av.Err is not nil.
func (av AnyValue) Bool() (bool, error) {
	if av.Err != nil {
		return false, av.Err
	}
	val, ok := av.Val.(bool)
	if !ok {
		return false, errs.ErrInvalidType("bool", av.Val)
	}
	return val, nil
}

// BoolOrDefault returns the stored value as a boolean, or a default value if there is an error.
// It reuses the Bool method to get the value and checks for any errors.
// If an error is encountered, it returns the provided default value.
//
// Parameters:
// - def bool: The default boolean value to return if there is an error.
// Returns:
// - bool: The boolean value stored in the struct or the default value if an error occurs.
func (av AnyValue) BoolOrDefault(def bool) bool {
	val, err := av.Bool()
	if err != nil {
		return def
	}
	return val
}

// JSONScan converts the stored byte slice (or string) to a JSON object and unmarshals it into the provided value.
// It first uses the AsBytes method to get the byte slice from the stored value.
// If successful, it unmarshals the byte slice into the provided value using json.Unmarshal.
//
// Parameters:
// - val any: The value to which the JSON data should be unmarshalled.
// Returns:
// - error: An optional error if something went wrong during conversion or unmarshalling.
func (av AnyValue) JSONScan(val any) error {
	data, err := av.AsBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, val)
}
