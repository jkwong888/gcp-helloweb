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
	"log"
	"net/http"
	"os"
	"strings"
)

func getMetaData(ctx context.Context, path string) string {
	metaDataURL := "http://metadata/computeMetadata/v1/"
	req, _ := http.NewRequest(
		"GET",
		metaDataURL+path,
		nil,
	)
	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	return string(makeRequest(req))
}

func makeRequest(r *http.Request) []byte {
	transport := http.Transport{DisableKeepAlives: true}
	client := &http.Client{Transport: &transport}
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
	return body
}

func main() {
	// use PORT environment variable, or default to 8080
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}

	// register hello function to handle all requests
	server := http.NewServeMux()
	server.HandleFunc("/", hello)

	// start the web server on port and accept requests
	log.Printf("Server listening on port %s", port)
	err := http.ListenAndServe(":"+port, server)
	log.Fatal(err)
}

// hello responds to the request with a plain-text "Hello, world" message.
func hello(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving request: %s", r.URL.Path)
	ctx := context.Background()

	host, _ := os.Hostname()
	zoneStr := getMetaData(ctx, "instance/zone")
	nodeName := getMetaData(ctx, "instance/hostname")
	region := getMetaData(ctx, "instance/attributes/cluster-location")
	clusterName := getMetaData(ctx, "instance/attributes/cluster-name")
	project := getMetaData(ctx, "project/project-id")
	//internalIP := getMetaData(ctx, "instance/network-interfaces/0/ip")

	zoneArr := strings.Split(zoneStr, "/")
	zone := zoneArr[len(zoneArr)-1]

	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: 1.0.0\n")
	fmt.Fprintf(w, "Hostname: %s\n", host)
	fmt.Fprintf(w, "Node Name: %s\n", nodeName)
	fmt.Fprintf(w, "Cluster Name: %s\n", clusterName)
	fmt.Fprintf(w, "Cluster Region: %s\n", region)
	fmt.Fprintf(w, "Zone: %s\n", zone)
	fmt.Fprintf(w, "Project: %s\n", project)
	//	fmt.Fprintf(w, "Internal IP: %s\n", internalIP)

}

// [END all]
