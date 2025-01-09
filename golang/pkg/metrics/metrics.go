package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)
/*
var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"path"},
)
*/

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"path", "status"},
)

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_time_seconds",
	Help: "Duration of HTTP requests.",
}, []string{"path"})

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func NewResponseWriterInterceptor(w http.ResponseWriter) (*responseWriterInterceptor) {
	return &responseWriterInterceptor{
		ResponseWriter: w,
		statusCode: http.StatusOK,
	}
}

func InitMetrics() {
	//prometheus.Register(totalRequests)
	prometheus.Register(responseStatus)
	prometheus.Register(httpDuration)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.RequestURI

		rw := NewResponseWriterInterceptor(w)

		// record time
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		defer timer.ObserveDuration()
		next.ServeHTTP(rw, r)

		// record status codes
		statusCode := rw.statusCode
		responseStatus.WithLabelValues(path, strconv.Itoa(statusCode)).Inc()

		// increment total requests
		//totalRequests.WithLabelValues(path).Inc()

	})
}

func MetricsHandler() (http.HandlerFunc) {
	// Stats exporter: Prometheus
	/*
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "helloweb",
	})
	if err != nil {
		zap.S().Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	view.RegisterExporter(pe)
	*/

	return promhttp.Handler().ServeHTTP

}
