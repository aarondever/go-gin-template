// Package telemetry installs the global OpenTelemetry tracer. Traces only:
// logs go through internal/logger, stamped with the current trace id.
package telemetry

import (
	"context"
	"fmt"

	"github.com/aarondever/go-gin-template/config"
	"github.com/aarondever/go-gin-template/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Init sets the global propagator and tracer provider and returns the
// flush-and-close for shutdown. Call it before anything it instruments.
//
// An empty Endpoint still installs a real provider, just without an exporter:
// the no-op one mints all-zero ids, leaving logs nothing to correlate on.
func Init(ctx context.Context, cfg config.OTELConfig) (func(context.Context) error, error) {
	// Set even with telemetry off: a service in the middle must pass the
	// incoming trace context on whether or not it records anything.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		)),
		// ParentBased, so a trace the caller sampled stays whole. At ratio 0 spans
		// still carry ids, they are just not recorded.
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	}

	if cfg.Endpoint != "" {
		// Export failures are asynchronous, so without this a collector refusing
		// spans looks like one receiving them.
		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			logger.Error("telemetry", logger.Err(err))
		}))

		// No dial here, so a collector that is down delays spans, not the boot.
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(cfg.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("telemetry: build exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}
