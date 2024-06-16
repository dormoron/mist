package security

import (
	"github.com/dormoron/mist"
	"net/http"
)

type MiddlewareBuilder struct {
	sp Provider
}

func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			sess, err := b.sp.Get(ctx)
			if err != nil {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			ctx.Set(CtxSessionKey, sess)
			next(ctx)
		}
	}
}
