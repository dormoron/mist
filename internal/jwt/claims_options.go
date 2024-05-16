package jwt

import (
	"github.com/dormoron/mist/utils"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// Options define the configuration for JWT token management.
//
// Parameters:
// - Expire: A time.Duration value indicating the expiration duration of the token.
// - EncryptionKey: A string value used for token encryption.
//
// Returns:
// - Options: This method returns an Options struct initialized with the provided parameters or defaults.
type Options struct {
	Expire        time.Duration     // Duration before a token expires.
	EncryptionKey string            // Key used for JWT encryption.
	DecryptKey    string            // Key used for JWT decryption, defaults to EncryptionKey if not provided.
	Method        jwt.SigningMethod // Method used to sign the JWT.
	Issuer        string            // Name or identifier of the issuer of the JWT.
	genIDFn       func() string     // Function to generate a unique ID for the JWT.
}

// InitOptions initializes and returns an Options struct with given parameters and additional options.
//
// Parameters:
// - expire: Duration before the token expires.
// - encryptionKey: Key used for token encryption.
// - opts: Optional functional parameters to customize the Options further.
//
// Returns:
// - Options: A struct containing configuration options for JWT token management.
func InitOptions(expire time.Duration, encryptionKey string,
	opts ...utils.Option[Options]) Options {
	dOpts := Options{
		Expire:        expire,
		EncryptionKey: encryptionKey,
		DecryptKey:    encryptionKey,          // Initialize DecryptKey with the same value as EncryptionKey by default.
		Method:        jwt.SigningMethodHS256, // Default signing method is HS256.
		// genIDFn is a function that generates a token identifier, initialized as an empty function by default.
		genIDFn: func() string { return "" },
	}
	utils.Apply[Options](&dOpts, opts...) // Apply any additional options provided.

	return dOpts
}

// WithDecryptKey is a functional option that sets the decryption key in Options.
//
// Parameters:
// - decryptKey: The key to be used for decryption.
//
// Returns:
// - utils.Option[Options]: A functional option to set the decryptKey field in Options.
func WithDecryptKey(decryptKey string) utils.Option[Options] {
	return func(o *Options) {
		o.DecryptKey = decryptKey
	}
}

// WithMethod is a functional option that sets the signing method in Options.
//
// Parameters:
// - method: JWT signing method.
//
// Returns:
// - utils.Option[Options]: A functional option to set the Method field in Options.
func WithMethod(method jwt.SigningMethod) utils.Option[Options] {
	return func(o *Options) {
		o.Method = method
	}
}

// WithIssuer is a functional option that sets the issuer in Options.
//
// Parameters:
// - issuer: The party who issues the JWT.
//
// Returns:
// - utils.Option[Options]: A functional option to set the Issuer field in Options.
func WithIssuer(issuer string) utils.Option[Options] {
	return func(o *Options) {
		o.Issuer = issuer
	}
}

// WithGenIDFunc is a functional option that sets the ID generation function in Options.
//
// Parameters:
// - fn: A function that generates a unique string ID.
//
// Returns:
// - utils.Option[Options]: A functional option to set the genIDFn field in Options.
func WithGenIDFunc(fn func() string) utils.Option[Options] {
	return func(o *Options) {
		o.genIDFn = fn
	}
}
