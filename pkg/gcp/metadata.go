package gcp

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	httpreq "helloworld-http/pkg/util"
)

func getMetaDataVal(path string, metadata map[string]interface{}) (interface{}) {
	//zap.S().Info("looking for path ", path, " in ", metadata)

	// string is a slash-delimited path to traverse down the json tree
	s := strings.SplitN(path, "/", 2)

	if len(s) == 0 {
		// empty string?
		return nil
	}

	p0 := s[0]
	br := metadata[p0]

	// check if the path fragment has a [] which indicates we expect an array
	rexp := regexp.MustCompile(`(?P<path>[a-zA-z]+)\[(?P<index>\d+)\]`)
	matched := rexp.MatchString(s[0])

	if matched {
		pathStr := "${path}"
		indexStr := "${index}"
		p0 = rexp.ReplaceAllString(s[0], pathStr)
		idx, _ := strconv.Atoi(rexp.ReplaceAllString(s[0], indexStr))

		if metadata[p0] == nil {
			return nil
		}

		brArray := metadata[p0].([]interface{})
		br = brArray[idx]

		// TODO: here we assume that it's an array of maps, what if it's just an array of strings or something?
	}


	if br == nil {
		//zap.S().Info("not found path: ", path)
		return nil
	}

	if len(s) == 1 {
		//zap.S().Info("found path ", path, " ", br)
		return br
	}

	// call myself on the remainder of the path
	return getMetaDataVal(s[1], br.(map[string]interface{}))
}

func GetMetaDataStrVal(path string, metadata map[string]interface{}) (*string) {
	val := getMetaDataVal(path, metadata)
	if val == nil {
		return nil
	}

	valStr := val.(string)
	return &valStr
}

func GetMetaDataArrVal(path string, metadata map[string]interface{}) ([]interface{}) {
	val := getMetaDataVal(path, metadata)
	if val == nil {
		return nil
	}

	valArr := val.([]interface{})
	return valArr
}

func GetMetaDataBoolVal(path string, metadata map[string]interface{}) (*bool) {
	val := getMetaDataVal(path, metadata)
	if val == nil {
		return nil
	}

	valBool := val.(bool)
	return &valBool
}

func GetProjectID(ctx context.Context) (*string, error) {
	metaDataURL := "http://metadata/computeMetadata/v1/project/project-id"
	req, err := http.NewRequest(
		"GET",
		metaDataURL,
		nil,
	)
	if (err != nil) {
		return nil, err
	}

	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	code, body := httpreq.MakeRequest(req)

	if code == 200 {
		bodyStr := string(body)
		var md map[string]interface{}
		json.Unmarshal([]byte(bodyStr), &md)
		zap.L().Info("Called metadata server", 
			zap.Any("metadata", md))


		return &bodyStr, nil
	}

	return nil, nil
}

func GetMetaData(ctx context.Context) (*string, error) {
	metaDataURL := "http://metadata/computeMetadata/v1/?recursive=true"
	req, err := http.NewRequest(
		"GET",
		metaDataURL,
		nil,
	)
	if (err != nil) {
		return nil, err
	}

	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	code, body := httpreq.MakeRequest(req)

	if code == 200 {
		bodyStr := string(body)
		var md map[string]interface{}
		json.Unmarshal([]byte(bodyStr), &md)
		zap.L().Debug("Called metadata server", 
			zap.Any("metadata", md))


		return &bodyStr, nil
	}

	return nil, nil
}
