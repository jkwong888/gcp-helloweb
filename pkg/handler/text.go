package handler

import (
	"fmt"
	"net/http"

	attrs "helloworld-http/pkg/attrs"
)

func (h *Handler) helloText(w http.ResponseWriter, attrs attrs.Payload) {
	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: %s\n", attrs.Version)
	fmt.Fprintf(w, "Request Headers:\n")
	for k, v := range attrs.Request.RequestHeaders {
		fmt.Fprintf(w, "  %s: %s\n", k, v)
	}

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