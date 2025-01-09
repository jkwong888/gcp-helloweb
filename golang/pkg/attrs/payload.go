package attrs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"

	trace "helloworld-http/pkg/trace"
	gcp "helloworld-http/pkg/gcp"
	util "helloworld-http/pkg/util"
)

const PAYLOAD_VERSION = "1.1.0"

type Payload struct {
	Version     	string `json:"version"`

	Request 	requestAttrs `json:"request"`

	NodeName       string `json:"nodename,omitempty"`
	Zone           string `json:"zone"`
	Project        string `json:"project"`
	ServiceAccount string `json:"serviceAccount"`

	Guest  guestAttrs  `json:"guest"`
	Client clientAttrs `json:"client"`

	Gae *gaeAttrs `json:"gae,omitempty"`
	Gce *gceAttrs `json:"gce,omitempty"`
	Gke *gkeAttrs `json:"gke,omitempty"`
	Run *runAttrs `json:"run,omitempty"`
	Cf  *cfAttrs  `json:"cf,omitempty"`
	K8s *k8sAttrs `json:"k8s,omitempty"`
}

type requestAttrs struct {
	RequestPath 	string `json:"requestPath"`
	RequestHeaders 	map[string][]string `json:"requestHeaders"`
}

type guestAttrs struct {
	Hostname    string `json:"hostname"`
	GuestIpAddr string `json:"guestIp"`
}

type clientAttrs struct {
	SourceAddr string  `json:"sourceAddr"`
	LbAddr     *string `json:"lbAddr,omitempty"`
}

type gaeAttrs struct {
	Region     string `json:"region"`
	InstanceId string `json:"instanceId"`
}

type gceAttrs struct {
	PrivateIpAddr string  `json:"privateIp"`
	MachineType   string  `json:"machineType,omitempty"`
	Preemptible   bool    `json:"preemptible"`
	MigName       *string `json:"migName,omitempty"`
}

type gkeAttrs struct {
	ClusterName   string `json:"clusterName"`
	ClusterRegion string `json:"clusterRegion"`
}

type k8sAttrs struct {
	PodName        string             `json:"podName"`
	PodIpAddr      string             `json:"podIpAddr"`
	Namespace      string             `json:"namespace"`
	ServiceAccount string             `json:"serviceAccount"`
	Labels         *map[string]string `json:"labels,omitempty"`
	NodeName       string             `json:"nodeName"`
	NodeIpAddr     string             `json:"nodeIpAddr"`
}

type runAttrs struct {
	Region     string `json:"region"`
	InstanceId string `json:"instanceId"`
}

type cfAttrs struct {
}

// GetLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
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

func getKeyValsFromDisk(filename string) *map[string]string {
    file, err := os.Open(filename)
    if err != nil {
        zap.S().Debug(err)
		return nil
    }
    defer file.Close()

	var labels = make(map[string]string)

    scanner := bufio.NewScanner(file)
    // optionally, resize scanner's capacity for lines over 64K, see next example
    for scanner.Scan() {
		text := scanner.Text()
		// split the text on =
		//zap.S().Info(text)

        //fmt.Println(scanner.Text())
		s := strings.SplitN(text, "=", 2)
		labels[s[0]] = strings.Trim(s[1], "\"")
    }

    if err := scanner.Err(); err != nil {
        zap.S().Error(err)
    }

	return &labels
}

func GetAllAttrs(ctx context.Context, r *http.Request, t *trace.TraceConfig) (Payload, error) {
	allVals := Payload{}

	span := t.StartTrace(ctx, "file io")
	vers, err := ioutil.ReadFile("version.txt")
	if err != nil {
		_ = fmt.Errorf("cannot find file, version.txt: %s", err)
	}
	span.End()

	allVals.Version = string(vers)

	allVals.Request = requestAttrs{}
	allVals.Request.RequestPath = r.URL.Path
	allVals.Request.RequestHeaders = r.Header

	span = t.StartTrace(ctx, "gcp metadata server")
	var metadata map[string]interface{}
	metadataStr, err := gcp.GetMetaData(ctx)
	if err != nil {
		zap.S().Errorf("Enable to retrieve metadata: %s", err)
		return allVals, err
	}
	span.End()

	// dump out the metadata to the zap.S().
	//var outJSON bytes.Buffer
	//json.Indent(&outJSON, []byte(*metadataStr), "", "  ")

	json.Unmarshal([]byte(*metadataStr), &metadata)

	nodeName := gcp.GetMetaDataStrVal("instance/hostname", metadata)
	if nodeName != nil {
		allVals.NodeName = *nodeName
	}

	zoneStr := gcp.GetMetaDataStrVal("instance/zone", metadata)
	if zoneStr != nil {
		zoneArr := strings.Split(*zoneStr, "/")
		zone := zoneArr[len(zoneArr)-1]

		allVals.Zone = zone
	}

	project := gcp.GetMetaDataStrVal("project/projectId", metadata)
	if project != nil {
		allVals.Project = *project
	}

	serviceAccount := gcp.GetMetaDataStrVal("instance/serviceAccounts/default/email", metadata)
	if serviceAccount != nil {
		allVals.ServiceAccount = *serviceAccount
	}

	/* Begin client attributes */

	// get the XFF header
	xffHdr := r.Header.Get("x-forwarded-for")

	if xffHdr != "" {
		ips := strings.Split(xffHdr, ",")

		// you can only trust the first two IPs, throw everything else away
		if len(ips) > 0 {
			allVals.Client.SourceAddr = ips[0]
		}

		// if we're in GCE and there's two IPs in this list, the second one is our LB
		if len(ips) > 1 {
			allVals.Client.LbAddr = &ips[1]
		}


	} else {
		// if xff header is not there, then it must be a direct client
		clientIpPort := r.RemoteAddr
		portSepIdx := strings.LastIndex(clientIpPort, ":")
		clientIp := clientIpPort[0:portSepIdx]
		allVals.Client.SourceAddr = clientIp
	}

	/* if we're in app engine, this header gets set, esp if we're not coming from a load balancer */
	xaecipHdr := r.Header.Get("x-appengine-user-ip")
	if xaecipHdr != "" {
		allVals.Client.SourceAddr = xaecipHdr
	}

	/* End client attributes */

	/* Begin guest attributes */
	host, _ := os.Hostname()
	allVals.Guest.Hostname = host

	localIp := getLocalIP()
	allVals.Guest.GuestIpAddr = localIp
	/* End guest attributes */

	scopes := gcp.GetMetaDataArrVal("instance/serviceAccounts/default/scopes", metadata)
	if util.ArrayContains(scopes, "https://www.googleapis.com/auth/appengine.apis") {
		/* Begin GAE attributes */
		// if we have appengine APIs in scope, we're probably in app engine
		if allVals.Gae == nil {
			allVals.Gae = &gaeAttrs{}
		}

		instanceId := gcp.GetMetaDataStrVal("instance/id", metadata)
		allVals.Gae.InstanceId = *instanceId

		regionStr := gcp.GetMetaDataStrVal("instance/region", metadata)
		rexp := regexp.MustCompile(`.*/regions/`)
		region := rexp.ReplaceAllString(*regionStr, "")
		allVals.Gae.Region = region
	/* End GAE attributes */
	} else if util.ArrayContains(scopes, "https://www.googleapis.com/auth/calendar") {
		/* Begin Run attributes */
		/* a guess that these types of scopes mean we're in cloud run, if we're not in app engine */
		// if we have appengine APIs in scope, we're probably in app engine
		if allVals.Run == nil {
			allVals.Run = &runAttrs{}
		}

		instanceId := gcp.GetMetaDataStrVal("instance/id", metadata)
		allVals.Run.InstanceId = *instanceId

		regionStr := gcp.GetMetaDataStrVal("instance/region", metadata)
		rexp := regexp.MustCompile(`.*/regions/`)
		region := rexp.ReplaceAllString(*regionStr, "")
		allVals.Run.Region = region
		/* End Run attributes */
	}

	/* Begin GCE attributes */
	machineType := gcp.GetMetaDataStrVal("instance/machineType", metadata)
	if machineType != nil {
		// assumption: all GCE machines will have the machine-type property
		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		rexp := regexp.MustCompile(`.*/machineTypes/`)
		machineTypeStr := rexp.ReplaceAllString(*machineType, "")
		allVals.Gce.MachineType = machineTypeStr
	}

	internalIP := gcp.GetMetaDataStrVal("instance/networkInterfaces[0]/ip", metadata)
	if internalIP != nil {
		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		allVals.Gce.PrivateIpAddr = *internalIP
	}

	createdBy := gcp.GetMetaDataStrVal("instance/attributes/createdBy", metadata)
	if createdBy != nil {
		rexp := regexp.MustCompile(`.*/instanceGroupManagers/`)
		migNameStr := rexp.ReplaceAllString(*createdBy, "")

		if allVals.Gce == nil {
			allVals.Gce = &gceAttrs{}
		}

		allVals.Gce.MigName = &migNameStr
	}

	preemptible := gcp.GetMetaDataStrVal("instance/scheduling/preemptible", metadata)
	if preemptible != nil && *preemptible == "TRUE" {
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
	clusterName := gcp.GetMetaDataStrVal("instance/attributes/clusterName", metadata)
	if clusterName != nil {
		if allVals.Gke == nil {
			allVals.Gke = &gkeAttrs{}
		}

		allVals.Gke.ClusterName = *clusterName
	}

	region := gcp.GetMetaDataStrVal("instance/attributes/clusterLocation", metadata)
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

	return allVals, nil
}