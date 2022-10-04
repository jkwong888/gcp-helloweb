package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
)

func NullHealthCheck() healthcheck.Check {
	return func() error {
		return nil
	}
}

func EnableHealthCheck(mux *http.ServeMux) {
	// add health check
	health := healthcheck.NewHandler()

	//metaDataURL := "http://metadata/computeMetadata/v1"
	metadataHostname := "metadata.google.internal"
	health.AddReadinessCheck(
		"upstream-dep-dns",
		healthcheck.DNSResolveCheck(metadataHostname, 500*time.Millisecond))
	health.AddReadinessCheck(
		"upstream-dep-tcp",
		healthcheck.TCPDialCheck(fmt.Sprintf("%v:80", metadataHostname), 500*time.Millisecond))

	health.AddLivenessCheck(
		"upstream-dep-dns",
		healthcheck.DNSResolveCheck(metadataHostname, 500*time.Millisecond))
	health.AddLivenessCheck(
		"upstream-dep-tcp",
		healthcheck.TCPDialCheck(fmt.Sprintf("%v:80", metadataHostname), 500*time.Millisecond))

	mux.HandleFunc("/healthz", health.ReadyEndpoint)

	/*
		// Trace exporter: Zipkin
		localEndpoint, err := openzipkin.NewEndpoint("ochttp_tutorial", "localhost:5454")
		if err != nil {
			log.Fatalf("Failed to create the local zipkinEndpoint: %v", err)
		}
		reporter := zipkinHTTP.NewReporter("http://localhost:9411/api/v2/spans")
		ze := zipkin.NewExporter(reporter, localEndpoint)
		trace.RegisterExporter(ze)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	*/
}
