package jwt

import (
	"github.com/dormoron/mist"
	"github.com/golang-jwt/jwt/v5"
)

// Manager is an interface for managing middleware, tokens, and claims. It is generic to allow different data types.
type Manager[T any] interface {
	// MiddlewareBuilder is a method that returns a pointer to an instance of MiddlewareBuilder.
	// This builder can be used to set up proper middleware for request handling.
	MiddlewareBuilder() *MiddlewareBuilder[T]

	// Refresh is a method to refresh the context of the middleware.
	// It could be used for updating/refreshing authentication or any other context-specific data.
	Refresh(ctx *mist.Context)

	// GenerateAccessToken is a method to generate a new access token from provided data.
	// The data type is dynamic and can be adjusted as needed. The function returns the generated token as a string and any possible error.
	GenerateAccessToken(data T) (string, error)

	// VerifyAccessToken verifies the provided JWT token and returns the associated claims or an error.
	// The 'opts' argument provides additional options to the jwt.Parser and is optional.
	VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	// GenerateRefreshToken is used to generate a new refresh token from the provided data.
	GenerateRefreshToken(data T) (string, error)

	// VerifyRefreshToken verifies the provided refresh token string and returns the associated claims or error.
	VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	// SetClaims is a method to set registered claims to the current context.
	// The 'claims' parameter represents the registered claims to be set.
	SetClaims(ctx *mist.Context, claims RegisteredClaims[T])
}

// RegisteredClaims is a struct to hold data and registered JWT claims.
// The 'T' makes it robust to hold various types of data.
type RegisteredClaims[T any] struct {
	// The Data portion of the claim can be of any type 'T' and it is denoted in JSON representation as "data".
	Data T `json:"data"`
	// RegisteredClaims from JWT are embedded to contain standard claims defined in JWT specifications.
	jwt.RegisteredClaims
}
