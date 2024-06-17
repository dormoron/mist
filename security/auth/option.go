package auth

import (
	"github.com/dormoron/mist/security/auth/kit"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// Options struct defines the configuration options for token management.
type Options struct {
	// Expire defines the duration after which the token expires.
	Expire time.Duration

	// EncryptionKey is used to encrypt data.
	EncryptionKey string

	// DecryptKey is used to decrypt data.
	DecryptKey string

	// Method is the JWT signing method used for token generation.
	Method jwt.SigningMethod

	// Issuer is the entity that issues the token.
	Issuer string

	// genIDFn is a function that generates an ID, used for token identification.
	genIDFn func() string
}

// InitOptions initializes an Options struct with default or provided values.
// Parameters:
// - expire: The duration token should be valid for ('time.Duration').
// - encryptionKey: The key used for encryption ('string').
// - opts: A variadic list of functional options for configuring the Options struct ('kit.Option[Options]').
// Returns:
// - Options: The initialized Options struct.
func InitOptions(expire time.Duration, encryptionKey string, opts ...kit.Option[Options]) Options {
	dOpts := Options{
		Expire:        expire,                      // Set token expiration duration.
		EncryptionKey: encryptionKey,               // Set the encryption key.
		DecryptKey:    encryptionKey,               // Set the decryption key (same as encryption key by default).
		Method:        jwt.SigningMethodHS256,      // Set default JWT signing method.
		genIDFn:       func() string { return "" }, // Set default ID generation function.
	}

	// Apply additional options provided by the user.
	kit.Apply[Options](&dOpts, opts...)

	return dOpts
}

// WithDecryptKey is a functional option to set a custom decryption key in Options.
// Parameters:
// - decryptKey: The custom decryption key to be used ('string').
// Returns:
// - kit.Option[Options]: A function that sets the decryption key in Options.
func WithDecryptKey(decryptKey string) kit.Option[Options] {
	return func(o *Options) {
		o.DecryptKey = decryptKey // Set the custom decryption key.
	}
}

// WithMethod is a functional option to set a custom JWT signing method in Options.
// Parameters:
// - method: The JWT signing method to be used ('jwt.SigningMethod').
// Returns:
// - kit.Option[Options]: A function that sets the JWT signing method in Options.
func WithMethod(method jwt.SigningMethod) kit.Option[Options] {
	return func(o *Options) {
		o.Method = method // Set the custom JWT signing method.
	}
}

// WithIssuer is a functional option to set a custom issuer in Options.
// Parameters:
// - issuer: The custom issuer entity ('string').
// Returns:
// - kit.Option[Options]: A function that sets the issuer in Options.
func WithIssuer(issuer string) kit.Option[Options] {
	return func(o *Options) {
		o.Issuer = issuer // Set the custom issuer.
	}
}

// WithGenIDFunc is a functional option to set a custom ID generation function in Options.
// Parameters:
// - fn: The function used to generate IDs ('func() string').
// Returns:
// - kit.Option[Options]: A function that sets the ID generation function in Options.
func WithGenIDFunc(fn func() string) kit.Option[Options] {
	return func(o *Options) {
		o.genIDFn = fn // Set the custom ID generation function.
	}
}
