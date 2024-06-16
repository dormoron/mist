package token

import (
	"github.com/dormoron/mist"
	"github.com/golang-jwt/jwt/v5"
)

type Manager[T any] interface {
	MiddlewareBuilder() *MiddlewareBuilder[T]

	GenerateAccessToken(data T) (string, error)

	VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	GenerateRefreshToken(data T) (string, error)

	VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error)

	SetClaims(ctx *mist.Context, claims RegisteredClaims[T])
}

type RegisteredClaims[T any] struct {
	Data T `json:"data"`
	jwt.RegisteredClaims
}
