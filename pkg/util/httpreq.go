package util

import (
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
	"go.opencensus.io/plugin/ochttp"
)

func MakeRequest(r *http.Request) (int, []byte) {
	//transport := http.Transport{DisableKeepAlives: true}
	octr := &ochttp.Transport{}
	client := &http.Client{Transport: octr}

	zap.S().Debugf("Calling: %v %v", r.Method,  r.URL.Path)
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