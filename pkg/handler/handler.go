package handler

import (
	"mime"
	"net/http"
	"strings"

	attrs "helloworld-http/pkg/attrs"
	trace "helloworld-http/pkg/trace"

	"go.uber.org/zap"
)

type Handler struct {
	logger zap.Logger
	tracer trace.TraceConfig
}

func InitHandler(logger zap.Logger, tracer trace.TraceConfig) (*Handler, error) {
	return &Handler{
		logger: logger,
		tracer: tracer,
	}, nil
}

// hello responds to the request with a plain-text "Hello, world" message.
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	//zap.S().Infof("Serving request: %v %s", r.Method, r.URL.Path)
	zap.L().Info("Serving request", 
		zap.String("method", r.Method), 
		zap.String("path", r.URL.Path))
	zap.L().Debug("Request Headers", 
		zap.Any("headers", r.Header))

	attrs, err := attrs.GetAllAttrs(ctx, r, h.tracer)
	if err != nil {
		zap.S().Fatalf("error marshalling to html: %s", err)
		http.Error(w, "Error marshalling to html: %s", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("accept")
	for _, v := range strings.Split(contentType, ",") {
		if v != "" {
			t, _, err := mime.ParseMediaType(v)
			if err != nil {
				zap.S().Warnf("error parsing accept header: %s", v)
				continue
			}

			if t == "application/json" {
				h.helloJSON(w, attrs)
				return
			}

			if t == "text/html" {
				h.helloHTML(w, attrs)
				return
			}

		
			//zap.S().Printf("mimetype: %s", t)

		}
	}

	h.helloText(w, attrs)

}