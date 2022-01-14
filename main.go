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
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"encoding/json"

	//	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/heptiolabs/healthcheck"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

const PAYLOAD_VERSION = "1.1.0"

type Payload struct {
	Version 		string

	NodeName 		string
	Zone 			string
	Project 		string
	ServiceAccount 	string

	Guest 			guestAttrs
	Client 			clientAttrs

	Gce 			*gceAttrs		`json:",omitempty"`
	Gke 			*gkeAttrs		`json:",omitempty"`
	Run 			*runAttrs		`json:",omitempty"`	
	Cf 				*cfAttrs		`json:",omitempty"`
}

type guestAttrs struct {
	Hostname 		string
}

type clientAttrs struct {
	SourceIpAddr 	string
	LbIpAddr 		*string			`json:",omitempty"`
}

type gceAttrs struct {
	PrivateIpAddr 	string
	MachineType 	string
	Preemptible 	bool
	MigName 		*string			`json:",omitempty"`
}

type gkeAttrs struct {
	ClusterName 	string
	ClusterRegion 	string
}

type runAttrs struct {
}

type cfAttrs struct {
}

func getMetaData(ctx context.Context, path string) *string {
	metaDataURL := "http://metadata/computeMetadata/v1/"
	req, _ := http.NewRequest(
		"GET",
		metaDataURL+path,
		nil,
	)
	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	code, body := makeRequest(req)

	if code == 200 {
		bodyStr := string(body)
		return &bodyStr
	}

	return nil
}

func makeRequest(r *http.Request) (int, []byte) {
	//transport := http.Transport{DisableKeepAlives: true}
	octr := &ochttp.Transport{}
	client := &http.Client{Transport: octr}
	resp, err := client.Do(r)
	if err != nil {
		message := "Unable to call backend: " + err.Error()
		panic(message)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		message := "Unable to read response body: " + err.Error()
		panic(message)
	}

	return resp.StatusCode, body
}

func enableObservabilityAndExporters(mux *http.ServeMux) {
	// Stats exporter: Prometheus
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "helloweb",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	view.RegisterExporter(pe)

	mux.Handle("/metrics", pe)

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

func NullHealthCheck() healthcheck.Check {
	return func() error {
		return nil
	}
}

func enableHealthCheck(mux *http.ServeMux) {
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

func getAllAttrs() Payload {
	allVals := Payload{}
	ctx := context.Background()

	vers, err := ioutil.ReadFile("version.txt")
	if err != nil {
		fmt.Errorf("cannot find file, version.txt: %s", err)
	}

	allVals.Version = string(vers)

	nodeName := getMetaData(ctx, "instance/hostname")
	if nodeName != nil {
		allVals.NodeName = *nodeName
	}

	zoneStr := getMetaData(ctx, "instance/zone")
	if zoneStr != nil {
		zoneArr := strings.Split(*zoneStr, "/")
		zone := zoneArr[len(zoneArr)-1]

		allVals.Zone = zone
	}

	project := getMetaData(ctx, "project/project-id")
	if project != nil {
		allVals.Project = *project
	}

	serviceAccount := getMetaData(ctx, "instance/service-accounts/default/email")
	if serviceAccount != nil {
		allVals.ServiceAccount = *serviceAccount
	}

	/* Begin guest attributes */
	host, _ := os.Hostname()
	allVals.Guest.Hostname = host
	/* End guest attributes */

	/* Begin GCE attributes */
	machineType := getMetaData(ctx, "instance/machine-type")
	if machineType != nil {
		// assumption: all GCE machines will have the machine-type property
		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		rexp := regexp.MustCompile(`.*/machineTypes/`)
		machineTypeStr := rexp.ReplaceAllString(*machineType, "")
		allVals.Gce.MachineType = machineTypeStr
	}
	
	internalIP := getMetaData(ctx, "instance/network-interfaces/0/ip")
	if internalIP != nil {
		allVals.Gce.PrivateIpAddr = *internalIP
	}

	createdBy := getMetaData(ctx, "instance/attributes/created-by")
	if createdBy != nil {
		rexp := regexp.MustCompile(`.*/instanceGroupManagers/`)
		migNameStr := rexp.ReplaceAllString(*createdBy, "")
		allVals.Gce.MigName = &migNameStr
	}

	preemptible := getMetaData(ctx, "instance/scheduling/preemptible")
	if preemptible != nil  && *preemptible == "TRUE" {
		allVals.Gce.Preemptible = true
	} else {
		allVals.Gce.Preemptible = false
	}

	/* End GCE attributes */

	/* Begin GKE attributes */
	clusterName := getMetaData(ctx, "instance/attributes/cluster-name")
	if clusterName != nil {
		if allVals.Gke == nil {
			allVals.Gke = &gkeAttrs{}
		}

		allVals.Gke.ClusterName = *clusterName
	}

	region := getMetaData(ctx, "instance/attributes/cluster-location")
	if region != nil {
		if allVals.Gke == nil {
			allVals.Gke = &gkeAttrs{}
		}

		allVals.Gke.ClusterName = *clusterName
	}

	/* End GKE attributes */

	return allVals
}

// helloJSON responds with json response
func helloJSON(w http.ResponseWriter, r *http.Request) {
	attrs := getAllAttrs()

	jsonObj, err := json.Marshal(attrs)

	if err != nil {
		log.Fatalf("error marshalling to json: %s", err)
		http.Error(w, "Error marshalling to json: %s", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s", string(jsonObj))

}

// hello responds to the request with a plain-text "Hello, world" message.
func hello(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving request: %s", r.URL.Path)

	contentType := r.Header.Get("accept")
	if contentType != "" {
		t, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			log.Fatalf("error parsing accept header: %s", contentType)
		}

		if t == "application/json" {
			helloJSON(w, r)
			return
		}
	}

	attrs := getAllAttrs()

	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: %s\n", attrs.Version)
	fmt.Fprintf(w, "Hostname: %s\n", attrs.Guest.Hostname)

	fmt.Fprintf(w, "Zone: %s\n", attrs.Zone)
	fmt.Fprintf(w, "Project: %s\n", attrs.Project)
	fmt.Fprintf(w, "Node FQDN: %s\n", attrs.NodeName)
	fmt.Fprintf(w, "Service Account: %s\n", attrs.ServiceAccount)

	//	fmt.Fprintf(w, "Internal IP: %s\n", internalIP)

	// if we're in a VM
	if attrs.Gce != nil {
		fmt.Fprintf(w, "Machine Type: %s\n", attrs.Gce.MachineType)
		fmt.Fprintf(w, "Internal IP Address: %s\n", attrs.Gce.PrivateIpAddr)
		fmt.Fprintf(w, "Preemptible: %t\n", attrs.Gce.Preemptible)

		if attrs.Gce.MigName != nil {
			fmt.Fprintf(w, "Managed Instance Group: %s\n", *attrs.Gce.MigName)
		}
	}

	// if we're in GKE 
	if attrs.Gke != nil {
		// get the cluster name
		fmt.Fprintf(w, "GKE Cluster Name: %s\n", attrs.Gke.ClusterName)
		fmt.Fprintf(w, "GKE Cluster Region: %s\n", attrs.Gke.ClusterRegion)
	}

}

func main() {

	// Firstly, we'll register ochttp Server views.
	if err := view.Register(ochttp.DefaultClientViews...); err != nil {
		log.Fatal("Failed to register client views for HTTP metrics: %v", err)
	}

	// use PORT environment variable, or default to 8080
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}

	mux := http.NewServeMux()

	// Enable observability, add /metrics endpoint to extract and examine stats.
	enableObservabilityAndExporters(mux)

	// Enable health check /healthz endpoint
	enableHealthCheck(mux)

	mux.HandleFunc("/", http.HandlerFunc(hello))

	h := &ochttp.Handler{Handler: mux}

	// start the web server on port and accept requests
	log.Printf("Server listening on port %s", port)
	err := http.ListenAndServe(":"+port, h)
	log.Fatal(err)
}
// [END all]
