package auth

import (
	"github.com/dormoron/mist"
	"github.com/golang-jwt/jwt/v5"
)

// Manager is a generic interface that manages tokens and claims for a given type T.
type Manager[T any] interface {
	// MiddlewareBuilder returns an instance of MiddlewareBuilder for the specified type T.
	// Returns:
	// - *MiddlewareBuilder[T]: A pointer to an instance of MiddlewareBuilder for the given type T.
	MiddlewareBuilder() *MiddlewareBuilder[T]

	// GenerateAccessToken generates an access token containing the given data of type T.
	// Parameters:
	// - data: The data to be included in the access token ('T').
	// Returns:
	// - string: The generated access token.
	// - error: An error if token generation fails.
	GenerateAccessToken(data T) (string, error)

	// VerifyAccessToken verifies the given access token and extracts the claims from it.
	// Parameters:
	// - token: The access token to be verified ('string').
	// - opts: Additional options for the JWT parser (variadic 'jwt.ParserOption').
	// Returns:
	// - RegisteredClaims[T]: The claims extracted from the verified token.
	// - error: An error if token verification fails.
	VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	// GenerateRefreshToken generates a refresh token containing the given data of type T.
	// Parameters:
	// - data: The data to be included in the refresh token ('T').
	// Returns:
	// - string: The generated refresh token.
	// - error: An error if token generation fails.
	GenerateRefreshToken(data T) (string, error)

	// VerifyRefreshToken verifies the given refresh token and extracts the claims from it.
	// Parameters:
	// - token: The refresh token to be verified ('string').
	// - opts: Additional options for the JWT parser (variadic 'jwt.ParserOption').
	// Returns:
	// - RegisteredClaims[T]: The claims extracted from the verified token.
	// - error: An error if token verification fails.
	VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	// SetClaims sets the provided claims in the context.
	// Parameters:
	// - ctx: The context where the claims are to be set ('*mist.Context').
	// - claims: The claims to be set in the context ('RegisteredClaims[T]').
	SetClaims(ctx *mist.Context, claims RegisteredClaims[T])
}

// RegisteredClaims is a generic struct that holds claims registered in a JWT, including user-defined data of type T.
type RegisteredClaims[T any] struct {
	Data                 T `json:"data"` // Custom data of type T associated with the registered claims.
	jwt.RegisteredClaims   // Embeds standard JWT registered claims.
}
