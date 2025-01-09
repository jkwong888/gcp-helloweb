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
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	chi "github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	metadata "helloworld-http/pkg/gcp"
	handler "helloworld-http/pkg/handler"
	health "helloworld-http/pkg/health"
	metrics "helloworld-http/pkg/metrics"
	trace "helloworld-http/pkg/trace"
	util "helloworld-http/pkg/util"
)



func main() {
	ctx := context.Background();

	cfg := zap.NewProductionConfig()
	a := zap.NewAtomicLevel()
	a.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	cfg.Level.SetLevel(a.Level())
	logger, _ := cfg.Build()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	project := os.Getenv("PROJECT_ID")
	if project == "" {
		// try to get it from the environment
		projectStr, err := metadata.GetProjectID(ctx)
		if err != nil {
			zap.S().Panicf("Failed to get metadata: %v", err)
		}

		project = *projectStr
	}

	zap.S().Infof("Project ID: %v", project)
	traceConfig, err := trace.InitTrace(ctx, project);
	if err != nil {
		zap.S().Panicf("Failed to initialize trace: %v", err)
	}
	defer traceConfig.Shutdown(ctx)


	startup_cpuloop, cpu_loop_exists := os.LookupEnv("STARTUP_CPULOOP_SECS")
	if (cpu_loop_exists) {
		busyloopSecs, err := strconv.Atoi(startup_cpuloop)
		if err != nil {
			zap.S().Panicf("Invalid value for STARTUP_CPULOOP_SECS: %v", startup_cpuloop)
		}

		util.BusyLoop(ctx, busyloopSecs)

	}

	handler, err := handler.InitHandler(*logger, traceConfig)
	if err != nil {
		zap.S().Panicf("Failed to initialize handler: %v", err)
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

	r := chi.NewRouter()

	r.Use(metrics.Middleware)
	r.Use(trace.Middleware)

	// Enable health check /healthz endpoint
	zap.S().Debug("Healthcheck available at /healthz")
	r.Get("/healthz", health.HealthCheckHandler())

	// add /metrics endpoint to extract and examine stats.
	metrics.InitMetrics()
	zap.S().Debug("Metrics available at /metrics")
	r.Get("/metrics", metrics.MetricsHandler())

	r.Post("/busyloop", http.HandlerFunc(handler.BusyLoop))

	// root handler which serves up responses
	r.Get("/*", http.HandlerFunc(handler.Hello))

	go func() {
		// start the web server on port and accept requests
		zap.S().Infof("Server listening on port %s", port)
		err := http.ListenAndServe(":"+port, r)
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

