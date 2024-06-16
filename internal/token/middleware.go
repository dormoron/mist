package token

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/token/kit"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

type MiddlewareBuilder[T any] struct {
	ignorePath func(path string) bool
	manager    *Management[T]
	nowFunc    func() time.Time
}

func newMiddlewareBuilder[T any](m *Management[T]) *MiddlewareBuilder[T] {
	return &MiddlewareBuilder[T]{
		manager: m,
		ignorePath: func(path string) bool {
			return false
		},
		nowFunc: m.nowFunc,
	}
}

func (m *MiddlewareBuilder[T]) IgnorePath(path ...string) *MiddlewareBuilder[T] {
	return m.IgnorePathFunc(staticIgnorePaths(path...))
}

func (m *MiddlewareBuilder[T]) IgnorePathFunc(fn func(path string) bool) *MiddlewareBuilder[T] {
	m.ignorePath = fn
	return m
}

func (m *MiddlewareBuilder[T]) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			if m.ignorePath(ctx.Request.URL.Path) {
				next(ctx)
				return
			}

			tokenStr := m.manager.extractTokenString(ctx)
			if tokenStr == "" {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			clm, err := m.manager.VerifyAccessToken(tokenStr,
				jwt.WithTimeFunc(m.nowFunc))
			if err != nil {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			m.manager.SetClaims(ctx, clm)
			next(ctx)
		}
	}

}

func staticIgnorePaths(paths ...string) func(path string) bool {
	s := kit.NewMapSet[string](len(paths))
	for _, path := range paths {
		s.Add(path)
	}
	return func(path string) bool {
		return s.Exist(path)
	}
}
