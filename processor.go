package traefik_change_response

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
)

const (
	ModeReplace = "replace"
	ModeKeep    = "keep"
	ModeAppend  = "append"
	ModePrepend = "prepend"
)

// changeResponse overrides response if status code in config matches
func changeResponse(wrapper *ResponseWriterWrapper, a *Plugin) {
	rw := wrapper.ResponseWriter

	// buffer current response values
	statusCode := wrapper.status
	headers := wrapper.ResponseWriter.Header()
	body := wrapper.body
	appliedOverride := false

	for _, o := range a.config.Overrides {
		// chain match by source code
		if slices.Contains(o.From, wrapper.status) {
			appliedOverride = true

			statusCode = o.To // can be rewritten multiple times

			for _, h := range o.RemoveHeaders {
				headers.Del(h) // remove previously set headers
			}

			for k, hv := range o.Headers {
				if _, ok := headers[k]; ok { // we have this header already
					headers.Del(k) // remove previously set headers
				}

				for _, h := range hv {
					headers.Add(k, h)
				}
			}

			// rewrite body
			switch o.Mode {
			case ModeKeep:
				// do nothing
			case ModeAppend:
				body.WriteString(o.Body)
			case ModePrepend:
				tmpBody := body.Bytes()
				body = bytes.NewBufferString(o.Body)
				body.Write(tmpBody)
			case ModeReplace, "": // replace is the default behavior
				body.Reset()
				body.WriteString(o.Body)
			default:
				panic("Unsupported override mode: " + o.Mode)
			}
		}
	}

	// Set modified content length
	headers.Set("Content-Length", strconv.Itoa(body.Len()))

	if appliedOverride && a.config.Debug {
		headers.Add("X-Applied-Plugin", a.name)
	}

	rw.WriteHeader(statusCode)

	// Write modified response
	if _, err := io.Copy(rw, body); err != nil {
		Alert("cannot write response body: " + err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	if a.config.Debug {
		Notify(fmt.Sprintf("writing body: [%d] %s", body.Len(), body.String()))
	}
}
