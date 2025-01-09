package handler

import (
	"fmt"
	"hash/fnv"
	"html/template"
	"net/http"

	"go.uber.org/zap"

	attrs "helloworld-http/pkg/attrs"
)

func stringToRGB(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	c := fmt.Sprintf("#%06X", h.Sum32()&0x00FFFFFF)
	return c
}

func (h *Handler) helloHTML(w http.ResponseWriter, attrs attrs.Payload) {
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
				background-color: {{ toRGB .Request.RequestPath }}
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
					<td colspan="2">{{ .Request.RequestPath }}</td>
				</tr>

				{{ if .NodeName }}
				<tr>
					<td>NodeName</td>
					<td colspan="2">{{ .NodeName }}</td>
				</tr>
				{{ end }}

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

				{{ if .Gae }}
				<tr>
					<th colspan="3">App Engine Attributes</th>
				</tr>
				<tr>
					<td>Instance ID</td>
					<td colspan="2">{{.Gae.InstanceId}}</td>
				</tr>
				<tr>
					<td>Region</td>
					<td colspan="2">{{.Gae.Region}}</td>
				</tr>
				{{ end }}

				{{ if .Run }}
				<tr>
					<th colspan="3">Cloud Run Attributes</th>
				</tr>
				<tr>
					<td>Instance ID</td>
					<td colspan="2">{{.Run.InstanceId}}</td>
				</tr>
				<tr>
					<td>Region</td>
					<td colspan="2">{{.Run.Region}}</td>
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
		zap.S().Fatalf("error marshalling to html: %s", err)
		http.Error(w, "Error marshalling to html: %s", http.StatusInternalServerError)
		return
	}

	t.Execute(w, attrs)
}