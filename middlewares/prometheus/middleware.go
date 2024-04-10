package prometheus

import (
	"github.com/dormoron/mist"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type MiddlewareBuilder struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

func (m MiddlewareBuilder) Build() mist.Middleware {
	vector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name,
		Help:      m.Help,
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, []string{"pattern", "method", "status"})

	prometheus.MustRegister(vector)
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			startTime := time.Now()
			defer func() {
				duration := time.Now().Sub(startTime).Microseconds()
				pattern := ctx.MatchedRoute
				if pattern == "" {
					pattern = "unknown"
				}
				vector.WithLabelValues(pattern, ctx.Request.Method, strconv.Itoa(ctx.RespStatusCode)).Observe(float64(duration))
			}()
			next(ctx)

		}
	}
}
