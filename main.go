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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"mime"
	"net"
	"net/http"

	//	"net/http/httptest"
	"html/template"
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
	RequestPath		string			`json:"requestPath"`

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
	K8s				*k8sAttrs		`json:"k8s,omitempty"`
}

type guestAttrs struct {
	Hostname 		string			`json:"hostname"`
	GuestIpAddr 	string			`json:"guestIp"`
}

type clientAttrs struct {
	SourceAddr 	string			`json:"sourceAddr"`
	LbAddr 		*string			`json:"lbAddr,omitempty"`
}

type gceAttrs struct {
	PrivateIpAddr 	string			`json:"privateIp"`
	MachineType 	string			`json:"machineType,omitempty"`
	Preemptible 	bool			`json:"preemptible"`
	MigName 		*string			`json:"migName,omitempty"`
}

type gkeAttrs struct {
	ClusterName 	string			`json:"clusterName"`
	ClusterRegion 	string			`json:"clusterRegion"`
}

type k8sAttrs struct {
	PodName			string				`json:"podName"`
	PodIpAddr		string				`json:"podIpAddr"`
	Namespace		string				`json:"namespace"`
	ServiceAccount	string				`json:"serviceAccount"`
	Labels			*map[string]string  `json:"labels,omitempty"`
	NodeName		string				`json:"nodeName"`
	NodeIpAddr		string				`json:"nodeIpAddr"`
}

type runAttrs struct {
}

type cfAttrs struct {
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, address := range addrs {
        // check the address type and if it is not a loopback the display it
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
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

func stringToRGB(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	c := fmt.Sprintf("#%06X", h.Sum32() & 0x00FFFFFF)
	return c
}

func getKeyValsFromDisk(filename string) *map[string]string {
    file, err := os.Open(filename)
    if err != nil {
        log.Error(err)
		return nil
    }
    defer file.Close()

	var labels = make(map[string]string)

    scanner := bufio.NewScanner(file)
    // optionally, resize scanner's capacity for lines over 64K, see next example
    for scanner.Scan() {
		text := scanner.Text()
		// split the text on =
		//log.Info(text)

        //fmt.Println(scanner.Text())
		s := strings.SplitN(text, "=", 2)
		labels[s[0]] = strings.Trim(s[1], "\"")
    }

    if err := scanner.Err(); err != nil {
        log.Error(err)
    }

	return &labels
}

func getAllAttrs(r *http.Request) Payload {
	allVals := Payload{}
	ctx := context.Background()

	vers, err := ioutil.ReadFile("version.txt")
	if err != nil {
		_ = fmt.Errorf("cannot find file, version.txt: %s", err)
	}

	allVals.Version = string(vers)
	allVals.RequestPath = r.URL.Path

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

	localIp := GetLocalIP()
	allVals.Guest.GuestIpAddr = localIp
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
		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		allVals.Gce.PrivateIpAddr = *internalIP
	}

	createdBy := getMetaData(ctx, "instance/attributes/created-by")
	if createdBy != nil {
		rexp := regexp.MustCompile(`.*/instanceGroupManagers/`)
		migNameStr := rexp.ReplaceAllString(*createdBy, "")

		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		allVals.Gce.MigName = &migNameStr
	}

	preemptible := getMetaData(ctx, "instance/scheduling/preemptible")
	if preemptible != nil  && *preemptible == "TRUE" {
		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		allVals.Gce.Preemptible = true
	} else {
		if allVals.Gce != nil {
			allVals.Gce.Preemptible = false
		}
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

		allVals.Gke.ClusterRegion = *region
	}

	/* End GKE attributes */

	/* Begin K8S Attributes -- should be passed from the Downward API*/
	if k8sNodeName := os.Getenv("K8S_NODE_NAME"); k8sNodeName != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.NodeName = k8sNodeName
	}

	if k8sNodeIp := os.Getenv("K8S_NODE_IP"); k8sNodeIp != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.NodeIpAddr = k8sNodeIp
	}


	if k8sPodName := os.Getenv("K8S_POD_NAME"); k8sPodName != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.PodName = k8sPodName
	}

	if k8sPodNamespace := os.Getenv("K8S_POD_NAMESPACE"); k8sPodNamespace != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.Namespace = k8sPodNamespace
	}

	if k8sPodIp := os.Getenv("K8S_POD_IP"); k8sPodIp != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.PodIpAddr = k8sPodIp
	}

	if k8sServiceAccount := os.Getenv("K8S_POD_SERVICE_ACCOUNT"); k8sServiceAccount != "" {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.ServiceAccount = k8sServiceAccount
	}

	// the pod labels should be mounted in /podinfo/labels
	labels := getKeyValsFromDisk("/podinfo/labels")
	if labels != nil {
		if allVals.K8s == nil {
			allVals.K8s = &k8sAttrs{}
		}

		allVals.K8s.Labels = labels

	}


	/* End K8S Attributes */

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

func helloHTML(w http.ResponseWriter, attrs Payload) {
    funcMap := template.FuncMap{
        // The name "inc" is what the function will be called in the template text.
        "inc": func(i int) int {
            return i + 1
        },
		"toRGB": func(s string) string {
			return stringToRGB(s)
		},
    }

	t := template.New("responseTemplate")

	htmlOut := `
<html>
	<head>
		<style>
			body {
				background-color: {{ toRGB .RequestPath }}
			}
			h1 {
				font-family: Arial, Helvetica, sans-serif;
			}
			table, th, td {
				border: 1px solid black;
				border-collapse: collapse;
			}
			table {
				min-width: 150px;
				max-width: 750px;
			}
			th, td {
				padding-top: 5px;
				padding-bottom: 5px;
				padding-left: 10px;
				padding-right: 10px;
				font-family: Arial, Helvetica, sans-serif;
			}
		</style>
	</head>
	<body>
		<div align="center">
			<h1>Hello, world!</h1>
			<div>
			<table>
				<tbody>
				<tr>
					<td>App Version</td>
					<td colspan="2">{{ .Version }}</td>
				</tr>
				<tr>
					<td>Request Path</td>
					<td colspan="2">{{ .RequestPath }}</td>
				</tr>

				<tr>
					<td>NodeName</td>
					<td colspan="2">{{ .NodeName }}</td>
				</tr>
				<tr>
					<td>Zone</td>
					<td colspan="2">{{ .Zone }}</td>
				</tr>
				<tr>
					<td>Project</td>
					<td colspan="2">{{ .Project }}</td>
				</tr>
				<tr>
					<td>Service Account</td>
					<td colspan="2">{{ .ServiceAccount }}</td>
				</tr>

				<tr>
					<th colspan="3">Guest Attributes</th>
				</tr>
				<tr>
					<td>Hostname</td>
					<td colspan="2">{{.Guest.Hostname}}</td>
				</tr>
				<tr>
					<td>IP Address</td>
					<td colspan="2">{{.Guest.GuestIpAddr}}</td>
				</tr>
				<tr>
					<th colspan="3">Client Attributes</th>
				</tr>
				<tr>
					<td>Source Address</td>
					<td colspan="2">{{.Client.SourceAddr}}
				</tr>
				{{ if .Client.LbAddr }}
				<tr>
					<td>Load Balancer Address</td>
					<td colspan="2">{{.Client.LbAddr}}</td>
				</tr>
				{{ end }}

				{{ if .Gce }}
				<tr>
					<th colspan="3"> Compute Engine Attributes</th>
				</tr>
				<tr>
					<td>Private IP Address</td>
					<td colspan="2">{{.Gce.PrivateIpAddr}}</td>
				</tr>
				<tr>
					<td>Machine Type</td>
					<td colspan="2">{{.Gce.MachineType}}</td>
				</tr>
				<tr>
					<td>Preemptible</td>
					<td colspan="2">{{.Gce.Preemptible}}</td>
				</td>
				<tr>
					<td>MIG Name</td>
					<td colspan="2">{{.Gce.MigName}}</td>
				</tr>
				{{ end }}

				{{ if .Gke }}
				<tr>
					<th colspan="3">GKE Attributes</th>
				</tr>
				<tr>
					<td>Cluster Name</td>
					<td colspan="2">{{ .Gke.ClusterName }}</td>
				</tr>
				<tr>
					<td>Cluster Region</td>
					<td colspan="2">{{ .Gke.ClusterRegion }}</td>
				</tr>

				{{ end }}

				{{ if .K8s }}
				<tr>
					<th colspan="3">Kubernetes Attributes</th>
				</tr>
				<tr>
					<td>Pod Name</td>
					<td colspan="2">{{ .K8s.PodName }}</td>
				</tr>
				<tr>
					<td>Pod IP</td>
					<td colspan="2">{{ .K8s.PodIpAddr }}</td>
				</tr>
				<tr>
					<td>Namespace</td>
					<td colspan="2">{{ .K8s.Namespace }}</td>
				</tr>
				<tr>
					<td>Service Account Name</td>
					<td colspan="2">{{ .K8s.ServiceAccount }}</td>
				</tr>

				{{ if .K8s.Labels }}
				<tr>
					<td rowSpan="{{ inc (len .K8s.Labels) }}" >Pod Labels</td>
				</tr>
				{{ range $k, $v := .K8s.Labels }}
				<tr>
					<td width="40%">{{ $k }}</td>
					<td width="60%" colspan="2">{{ $v }}</td>
				</tr>
					{{end}}
				</tr>
				{{ end }}

				<tr>
					<td>Node Name</td>
					<td colspan="2">{{ .K8s.NodeName }}</td>
				</tr>
				<tr>
					<td>Node IP</td>
					<td colspan="2">{{ .K8s.NodeIpAddr }}</td>
				</tr>

				{{ end }}

				</tbody>
			</table>
			</div>
		</div>
	</body>
</html>
	`

	t, err := t.Funcs(funcMap).Parse(htmlOut)
	if err != nil {
		log.Fatalf("error marshalling to html: %s", err)
		http.Error(w, "Error marshalling to html: %s", http.StatusInternalServerError)
		return
	}

	t.Execute(w, attrs)
}

func helloText(w http.ResponseWriter, attrs Payload) {
	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: %s\n", attrs.Version)
	fmt.Fprintf(w, "Hostname: %s\n", attrs.Guest.Hostname)
	fmt.Fprintf(w, "Local IP Address: %s\n", attrs.Guest.GuestIpAddr)

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

	// if we're in a k8s pod 
	if attrs.K8s != nil {
		fmt.Fprintf(w, "Kubernetes Pod Name: %s\n", attrs.K8s.PodName)
		fmt.Fprintf(w, "Kubernetes Pod IP Address: %s\n", attrs.K8s.PodIpAddr)
		fmt.Fprintf(w, "Kubernetes Namespace: %s\n", attrs.K8s.Namespace)
		fmt.Fprintf(w, "Kubernetes ServiceAccount: %s\n", attrs.K8s.ServiceAccount)


		if attrs.K8s.Labels != nil {
			fmt.Fprintf(w, "Kubernetes Pod Labels:\n")
			for k, v := range *attrs.K8s.Labels {
				fmt.Fprintf(w, "  %s: %s\n", k, v)
			}
		}

		fmt.Fprintf(w, "Kubernetes Node Name: %s\n", attrs.K8s.NodeName)
		fmt.Fprintf(w, "Kubernetes Node IP Address: %s\n", attrs.K8s.NodeIpAddr)

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

			if t == "text/html" {
				helloHTML(w, attrs)
				return
			}

		
			//log.Printf("mimetype: %s", t)

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
