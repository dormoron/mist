package opentelemetry

import (
	"github.com/dormoron/mist"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/dormoron/mist/middleware/opentelemetry"

// MiddlewareBuilder is a struct that aids in constructing middleware with tracing capabilities.
// It holds a reference to a Tracer instance which will be used to trace the flow of HTTP requests.
type MiddlewareBuilder struct {
	Tracer trace.Tracer // Tracer is an interface that abstracts the tracing functionality.
	// This tracer will be used to create spans for the structured monitoring of
	// application's request flows and performance.
}

// Build is a method attached to the MiddlewareBuilder struct. This method initializes
// and returns a Tracing middleware that can be used in the mist HTTP framework.
// This middleware is responsible for starting a new span for each incoming HTTP request,
// sets various attributes related to the request and ensures that the span is ended
// properly after the request is handled.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// Check if the Tracer attribute of MiddlewareBuilder has been set.
	// If not, initialize it with a default tracer from the OpenTelemetry global TracerProvider
	// using the specified instrumentation name.
	if m.Tracer == nil {
		m.Tracer = otel.GetTracerProvider().Tracer(instrumentationName)
	}

	// Return an anonymous function matching the mist middleware signature.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// This anonymous function is the actual middleware being executed per request.
		return func(ctx *mist.Context) {
			// Extract the current request context from the incoming HTTP request.
			reqCtx := ctx.Request.Context()

			// Inject distributed tracing headers into the request context.
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx, propagation.HeaderCarrier(ctx.Request.Header))

			// Start a new span with the request context, using the name "unknown" as a placeholder
			// until the actual route is matched.
			_, span := m.Tracer.Start(reqCtx, "unknown")

			// Defer the end of the span till after the request is handled.
			// This ensures the following code runs after the next handlers are completed,
			// right before exiting the middleware function.
			defer func() {
				// If the route was matched, name the span after the matched route.
				span.SetName(ctx.MatchedRoute)

				// Set additional attributes to the span, such as the HTTP status code.
				span.SetAttributes(attribute.Int("http.status", ctx.RespStatusCode))

				// End the span. This records the span's information and exports it to any configured telemetry systems.
				span.End()
			}()

			// Before proceeding, add additional HTTP-related information to the span,
			// such as the HTTP method, full URL, URL scheme, and host.
			span.SetAttributes(attribute.String("http.method", ctx.Request.Method))
			span.SetAttributes(attribute.String("http.url", ctx.Request.URL.String()))
			span.SetAttributes(attribute.String("http.scheme", ctx.Request.URL.Scheme))
			span.SetAttributes(attribute.String("http.host", ctx.Request.Host))

			// Update the request's context to include tracing context.
			ctx.Request = ctx.Request.WithContext(reqCtx)

			// Call the next function in the middleware chain with the updated context.
			next(ctx)
		}
	}
}
