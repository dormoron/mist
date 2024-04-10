package accesslog

import (
	"encoding/json"
	"github.com/dormoron/mist"
)

type MiddlewareBuilder struct {
	logFunc func(log string)
}

func (b *MiddlewareBuilder) LogFunc(fn func(log string)) *MiddlewareBuilder {
	b.logFunc = fn
	return b
}
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			defer func() {
				log := accessLog{
					Host:       ctx.Request.Host,
					Route:      ctx.MatchedRoute,
					HTTPMethod: ctx.Request.Method,
					Path:       ctx.Request.URL.Path,
				}
				data, _ := json.Marshal(log)
				b.logFunc(string(data))
			}()
			next(ctx)
		}
	}
}

type accessLog struct {
	Host       string `json:"host,omitempty"`
	Route      string `json:"route,omitempty"`
	HTTPMethod string `json:"http_method,omitempty"`
	Path       string `json:"path,omitempty"`
}
