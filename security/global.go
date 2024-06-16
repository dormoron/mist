package security

import (
	"github.com/dormoron/mist"
)

const CtxSessionKey = "_session"

var defaultProvider Provider

func NewSession(ctx *mist.Context, uid int64,
	jwtData map[string]string,
	sessData map[string]any) (Session, error) {
	return defaultProvider.NewSession(
		ctx,
		uid,
		jwtData,
		sessData,
	)
}

func Get(ctx *mist.Context) (Session, error) {
	return defaultProvider.Get(ctx)
}

func SetDefaultProvider(sp Provider) {
	defaultProvider = sp
}

func DefaultProvider() Provider {
	return defaultProvider
}

func CheckLoginMiddleware() mist.Middleware {
	return (&MiddlewareBuilder{sp: defaultProvider}).Build()
}

func RenewAccessToken(ctx *mist.Context) error {
	return defaultProvider.RenewAccessToken(ctx)
}

func UpdateClaims(ctx *mist.Context, claims Claims) error {
	return defaultProvider.UpdateClaims(ctx, claims)
}
