/**
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// [START all]
package main

import (
	"mime"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	//	"net/http/httptest"
	"os"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	attrs "helloworld-http/pkg/attrs"
	"helloworld-http/pkg/health"
	"helloworld-http/pkg/metrics"
	resp "helloworld-http/pkg/response"
)

// hello responds to the request with a plain-text "Hello, world" message.
func hello(w http.ResponseWriter, r *http.Request) {
	//zap.S().Infof("Serving request: %v %s", r.Method, r.URL.Path)
	zap.L().Info("Serving request", 
		zap.String("method", r.Method), 
		zap.String("path", r.URL.Path))
	zap.L().Debug("Request Headers", 
		zap.Any("headers", r.Header))

	attrs, err := attrs.GetAllAttrs(r)
	if err != nil {
		zap.S().Fatalf("error marshalling to html: %s", err)
		http.Error(w, "Error marshalling to html: %s", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("accept")
	for _, v := range strings.Split(contentType, ",") {
		if v != "" {
			t, _, err := mime.ParseMediaType(v)
			if err != nil {
				zap.S().Warnf("error parsing accept header: %s", v)
				continue
			}

			if t == "application/json" {
				resp.HelloJSON(w, attrs)
				return
			}

			if t == "text/html" {
				resp.HelloHTML(w, attrs)
				return
			}

		
			//zap.S().Printf("mimetype: %s", t)

		}
	}

	resp.HelloText(w, attrs)

}

func main() {

	cfg := zap.NewProductionConfig()
	a := zap.NewAtomicLevel()
	a.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	cfg.Level.SetLevel(a.Level())
	logger, _ := cfg.Build()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// Firstly, we'll register ochttp Server views.
	if err := view.Register(ochttp.DefaultClientViews...); err != nil {
		zap.S().Panicf("Failed to register client views for HTTP metrics: %v", err)
	}

	// use PORT environment variable, or default to 8080
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, 
		os.Interrupt, 
		syscall.SIGINT, 
		syscall.SIGTERM, 
		syscall.SIGQUIT)

	mux := http.NewServeMux()

	// Enable observability, add /metrics endpoint to extract and examine stats.
	metrics.EnableObservabilityAndExporters(mux)

	// Enable health check /healthz endpoint
	health.EnableHealthCheck(mux)

	// root handler which serves up responses
	mux.HandleFunc("/", http.HandlerFunc(hello))

	h := &ochttp.Handler{Handler: mux}

	go func() {
		// start the web server on port and accept requests
		zap.S().Infof("Server listening on port %s", port)
		err := http.ListenAndServe(":"+port, h)
		if err != nil {
			zap.S().Fatalf("Error listening: %v", err)
		}
	}()

	sig := <-sigCh
	switch sig {
	case os.Interrupt:
		zap.S().Info("Received SIGINT, terminating ...")
		os.Exit(0)
	case syscall.SIGTERM:
		zap.S().Info("Received SIGTERM, terminating ...")
		os.Exit(0)
	case syscall.SIGQUIT:
		zap.S().Info("Received SIGQUIT, terminating ...")
		os.Exit(0)
	}
	
}
// [END all]

