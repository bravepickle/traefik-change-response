package traefik_change_response

import (
	"bytes"
	"net/http"
)

// ResponseWriterWrapper captures the response body
type ResponseWriterWrapper struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

// WriteHeader Override WriteHeader to capture status code
func (rw *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rw.status = statusCode
	//rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseWriterWrapper) Write(data []byte) (int, error) {
	return rw.body.Write(data)
}
