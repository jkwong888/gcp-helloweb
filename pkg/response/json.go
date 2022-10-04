package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	attrs "helloworld-http/pkg/attrs"
)

// helloJSON responds with json response
func HelloJSON(w http.ResponseWriter, attrs attrs.Payload) {
	jsonObj, err := json.Marshal(attrs)

	if err != nil {
		zap.S().Fatalf("error marshalling to json: %s", err)
		http.Error(w, "Error marshalling to json: %s", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s", string(jsonObj))
}