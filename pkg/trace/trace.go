package trace

import (
	"context"
	//"fmt"
	"log"
	"net/http"
	//"strings"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	gcppropagator "github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const name = "helloworld-http"

type TraceConfig struct {
	TracerProvider sdktrace.TracerProvider
}

type Span struct {
	trace.Span
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//method = r.Method
		path := r.RequestURI

		if path == "/healthz" || path == "/metrics" {
			// don't send traces for health or metrics
			next.ServeHTTP(w, r)
			return
		}

		handler := otelhttp.NewHandler(
			next,
			path,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		)

		// add the traceId and spanId to the context for logger to read later
		//newCtx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		handler.ServeHTTP(w, r)
	})

		
		
	/*
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//method := rctx.RouteMethod
		path := r.RequestURI

		// import the trace if it's passed in
		traceCtx, found := r.Header["X-Cloud-Trace-Context"]
		if found {
			//zap.S().Info("Found trace header ", traceId)
			w.Header().Add("X-Cloud-Trace-Context", strings.Join(traceCtx, ","))
		}
		
		// start a new span under the trace
		tracer := otel.Tracer(name)
		newCtx, span := tracer.Start(r.Context(), fmt.Sprintf("%v", path))
		defer span.End()

		req := r.WithContext(newCtx)

		//route := mux.CurrentRoute(r)
		//path, _ := route.GetPathTemplate()

		next.ServeHTTP(w, req)

	})
	*/

}

func InitTrace(ctx context.Context, projectID string) (*TraceConfig, error) {
	exporter, err := texporter.New(texporter.WithProjectID(projectID))
	if err != nil {
		log.Fatalf("texporter.New: %v", err)
		return nil, err
	}

	// Identify your application using resource detection
	res, err := resource.New(ctx,
			// Use the GCP resource detector to detect information about the GCP platform
			resource.WithDetectors(gcp.NewDetector()),
			// Keep the default detectors
			resource.WithTelemetrySDK(),
			// Add your own custom attributes to identify your application
			resource.WithAttributes(
				semconv.ServiceNameKey.String("helloworld-http"),
			),
	)
	if err != nil {
		log.Fatalf("resource.New: %v", err)
		return nil, err
	}

	// Create trace provider with the exporter.
	//
	// By default it uses AlwaysSample() which samples all traces.
	// In a production environment or high QPS setup please use
	// probabilistic sampling.
	// Example:
	//   tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.0001)), ...)
	tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
	)
	defer tp.ForceFlush(ctx) // flushes any pending spans
	otel.SetTracerProvider(tp)

	compositePropagator := propagation.NewCompositeTextMapPropagator(
		gcppropagator.CloudTraceFormatPropagator{},
        propagation.TraceContext{}, 
        propagation.Baggage{})
        
  	otel.SetTextMapPropagator(compositePropagator)

	return &TraceConfig{
		*tp,
	}, nil
	
}

func (t *TraceConfig) Shutdown(ctx context.Context) {
	_ = t.TracerProvider.Shutdown(ctx);
}

func (t *TraceConfig) StartTrace(ctx context.Context, spanName string) (Span) {
	ctx, span := otel.Tracer(name).Start(ctx, spanName)

	return Span{
		Span: span,
	}
}