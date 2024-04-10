package recover

import "github.com/dormoron/mist"

type MiddlewareBuilder struct {
	StatusCode int
	Data       []byte
	Log        func(ctx *mist.Context)
}

func (m MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			defer func() {
				if r := recover(); r != nil {
					ctx.RespData = m.Data
					ctx.RespStatusCode = m.StatusCode
					m.Log(ctx)
				}
			}()
			next(ctx)
		}
	}
}
