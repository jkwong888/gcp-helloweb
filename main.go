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
	Version 		string			`json:"version"`

	NodeName 		string			`json:"nodename"`
	Zone 			string			`json:"zone"`
	Project 		string			`json:"project"`
	ServiceAccount 	string			`json:"serviceAccount"`

	Guest 			guestAttrs		`json:"guest"`
	Client 			clientAttrs		`json:"client"`

	Gce 			*gceAttrs		`json:"gce,omitempty"`
	Gke 			*gkeAttrs		`json:"gke,omitempty"`
	Run 			*runAttrs		`json:"run,omitempty"`	
	Cf 				*cfAttrs		`json:"cf,omitempty"`
}

type guestAttrs struct {
	Hostname 		string			`json:"hostname"`
}

type clientAttrs struct {
	SourceAddr 	string			`json:"sourceAddr"`
	LbAddr 		*string			`json:"lbAddr,omitempty"`
}

type gceAttrs struct {
	PrivateIpAddr 	string			`json:"privateIp"`
	MachineType 	string			`json:"machineType"`
	Preemptible 	bool			`json:"preemptible"`
	MigName 		*string			`json:"migName,omitempty"`
}

type gkeAttrs struct {
	ClusterName 	string			`json:"clusterName"`
	ClusterRegion 	string			`json:"clusterRegion"`
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

func getAllAttrs(r *http.Request) Payload {
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

	/* Begin client attributes */

	// get the XFF header
	xffHdr := r.Header.Get("x-forwarded-for")

	if xffHdr != "" {
		ips := strings.Split(xffHdr, ",")

		// you can only trust the first two IPs, throw everything else away
		if len(ips) > 1 {
			allVals.Client.SourceAddr = ips[0]
			allVals.Client.LbAddr = &ips[1]
		}
	} else {
		// if xff header is not there, then it must be a direct client
		clientIpPort := r.RemoteAddr
		portSepIdx := strings.LastIndex(clientIpPort, ":")
		clientIp := clientIpPort[0:portSepIdx]
		allVals.Client.SourceAddr = clientIp
	}

	/* End client attributes */

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
func helloJSON(w http.ResponseWriter, attrs Payload) {
	jsonObj, err := json.Marshal(attrs)

	if err != nil {
		log.Fatalf("error marshalling to json: %s", err)
		http.Error(w, "Error marshalling to json: %s", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s", string(jsonObj))
}

func helloText(w http.ResponseWriter, attrs Payload) {
	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: %s\n", attrs.Version)
	fmt.Fprintf(w, "Hostname: %s\n", attrs.Guest.Hostname)

	fmt.Fprintf(w, "Zone: %s\n", attrs.Zone)
	fmt.Fprintf(w, "Project: %s\n", attrs.Project)
	fmt.Fprintf(w, "Node FQDN: %s\n", attrs.NodeName)
	fmt.Fprintf(w, "Service Account: %s\n", attrs.ServiceAccount)


	fmt.Fprintf(w, "Client Addr: %s\n", attrs.Client.SourceAddr)
	if attrs.Client.LbAddr != nil {
		fmt.Fprintf(w, "LB Addr: %s\n", *attrs.Client.LbAddr)
	}

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

// hello responds to the request with a plain-text "Hello, world" message.
func hello(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving request: %s", r.URL.Path)

	attrs := getAllAttrs(r)

	contentType := r.Header.Get("accept")
	for _, v := range strings.Split(contentType, ",") {
		if v != "" {
			t, _, err := mime.ParseMediaType(v)
			if err != nil {
				log.Warnf("error parsing accept header: %s", v)
				continue
			}

			if t == "application/json" {
				helloJSON(w, attrs)
				return
			}
		}
	}

	helloText(w, attrs)

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
