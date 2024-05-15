package prometheus

import (
	"github.com/dormoron/mist"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

// MiddlewareBuilder is a struct that holds metadata for identifying and describing a set
// of middleware. This metadata includes details that are typically used for logging, monitoring,
// or other forms of introspection. The struct is not a middleware itself, but rather a
// collection of descriptive fields that may be associated with middleware operations.
type MiddlewareBuilder struct {
	Namespace string // Namespace is a top-level categorization that groups related subsystems. It's meant to prevent collisions between different subsystems.
	Subsystem string // Subsystem is a second-level categorization beneath namespace that allows for grouping related functionalities.
	Name      string // Name is the individual identifier for a specific middleware component. It should be unique within the namespace and subsystem.
	Help      string // Help is a descriptive string that provides insights into what the middleware does or is used for. It may be exposed in monitoring tools or documentation.
}

func InitMiddlewareBuilder(namespace string, subsystem string, name string, help string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}
}

// Build constructs and returns a new prometheus monitoring middleware for use within the mist framework.
// The MiddlewareBuilder receiver attaches the newly built prometheus metric collection functionality to mist Middleware.
// The metrics collected are specifically SummaryVec which help in observing the request latency distribution.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// A SummaryVec is created with the necessary prometheus options including the provided namespace, subsystem,
	// and the name from the MiddlewareBuilder fields. Additionally, the helper message and specific objectives for quantiles are set.
	vector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.Namespace, // Uses the namespace provided in the MiddlewareBuilder
		Subsystem: m.Subsystem, // Uses the subsystem provided in the MiddlewareBuilder
		Name:      m.Name,      // Uses the name provided in the MiddlewareBuilder
		Help:      m.Help,      // Uses the help message provided in the MiddlewareBuilder
		// Objectives is a map defining the quantile rank and the allowable error.
		// This allows us to calculate, e.g., the 50th percentile (median) with 1% error.
		Objectives: map[float64]float64{
			0.5:   0.01,   // 50th percentile (median)
			0.75:  0.01,   // 75th percentile
			0.90:  0.01,   // 90th percentile
			0.99:  0.001,  // 99th percentile, with a smaller error of 0.1%
			0.999: 0.0001, // 99.9th percentile, with an even smaller error
		},
		// Labels are predefined which we will later assign values for each observation.
	}, []string{"pattern", "method", "status"})

	// Register the SummaryVec to prometheus; MustRegister panics if this fails
	prometheus.MustRegister(vector)

	// Return a new middleware function which will be called during request processing in the mist framework
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			startTime := time.Now() // Record the start time when the request processing begins
			// Defer a function to ensure it's executed after the main middleware logic.
			// It measures the time taken to process the request and records it as an observation in the SummaryVec.
			defer func() {
				// Calculate the duration since the start time in microseconds
				duration := time.Now().Sub(startTime).Microseconds()

				// Retrieve the matched route pattern from the context, use "unknown" as a default
				pattern := ctx.MatchedRoute
				if pattern == "" {
					pattern = "unknown"
				}

				// Use the pattern, HTTP method, and status code as labels to observe the request duration
				// Float64 conversion is necessary to record the duration in the correct format
				vector.WithLabelValues(pattern, ctx.Request.Method, strconv.Itoa(ctx.RespStatusCode)).Observe(float64(duration))
			}()

			// Proceed with the next middleware or final handler in the chain
			next(ctx)
		}
	}
}
