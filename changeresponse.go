// Package traefik_change_response plugin.
package traefik_change_response

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
)

// Notify sends info message to the main application notification
var Notify = func(msg string) {
	os.Stdout.WriteString("[changeresponse.DEBUG] " + msg)
}

// Alert sends error message to the main application notification
var Alert = func(msg string) {
	os.Stderr.WriteString("[changeresponse.ERROR] " + msg)
}

// Config the plugin configuration.
type Config struct {
	Overrides []Override `json:"overrides"`
	Debug     bool       `json:"debug,omitempty"` // debug plugin - verbose mode
}

// Override is a single override rule for the plugin
type Override struct {
	// From list of HTTP status codes to match against to apply this override rule. Required
	From []int `json:"from"`

	// To status code to substitute the initial one. Required
	To int `json:"to"`

	// Headers sets defined headers in response. Optional
	Headers http.Header `json:"headers,omitempty"`

	// RemoveHeaders removes upstream response headers if matched override. Optional
	RemoveHeaders []string `json:"removeHeaders,omitempty"`

	// Body overrides body contents - based on mode rule selected. Optional
	Body string `json:"body,omitempty"`

	// Mode Replaces body in a specified manner. Optional
	// Allowed:
	//   replace (default) - replace body contents
	//   keep - keep body as it is
	//   append - append extra body contents to the end
	//   prepend - prepend extra body contents
	Mode string `json:"mode,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Plugin a plugin main entity.
type Plugin struct {
	next   http.Handler
	name   string
	config *Config
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config must be defined")
	}

	if len(config.Overrides) == 0 {
		return nil, fmt.Errorf("at least one override rule is required")
	}

	if config.Debug {
		Notify(fmt.Sprintf("defined config %s: %v", name, config))
	}

	return &Plugin{
		next:   next,
		name:   name,
		config: config,
	}, nil
}

// ServeHTTP processes requests/responses as a middleware
func (a *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrapper := &ResponseWriterWrapper{body: &bytes.Buffer{}, ResponseWriter: rw}
	a.next.ServeHTTP(wrapper, req)
	changeResponse(wrapper, a)
}
