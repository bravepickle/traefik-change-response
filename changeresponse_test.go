package traefik_change_response_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	changeresponse "github.com/bravepickle/traefik-change-response"
)

type inputDataset struct {
	name            string // dataset name
	config          changeresponse.Config
	responseCode    int
	responseHeaders http.Header
	responseBody    string
}

type ChangeResponseDataset struct {
	input           inputDataset
	expectedCode    int
	expectedHeaders http.Header
	expectedBody    string
}

type fatalNotifier interface {
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
}

func TestChangeResponse(t *testing.T) {
	datasets := []ChangeResponseDataset{
		{
			input: inputDataset{
				name: "basic",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{503},
							To:   500,
							Headers: http.Header{
								"Content-Type": []string{"application/json"},
								"X-Foo":        []string{"bar", "baz"},
							},
							Body: `{"status": "ok", "msg": "Test override"}`,
						},
					},
				},
				responseCode: 503,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"X-Foo":        []string{"initial"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Client response body",
			},
			expectedCode: 500,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"application/json"},
				"X-Foo":          []string{"bar", "baz"},
				"Content-Length": []string{strconv.Itoa(len(`{"status": "ok", "msg": "Test override"}`))},
			},
			expectedBody: `{"status": "ok", "msg": "Test override"}`,
		},
		{
			input: inputDataset{
				name: "blank response",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   204,
							Headers: http.Header{
								"Content-Type": []string{"application/json"},
							},
							Mode: "replace",
							Body: "",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Client response body",
			},
			expectedCode: 204,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"0"},
			},
			expectedBody: "",
		},
		{
			input: inputDataset{
				name: "rules mismatch",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{503},
							To:   500,
							Headers: http.Header{
								"Content-Type": []string{"application/json"},
								"X-Foo":        []string{"bar", "baz"},
							},
							Body: `{"status": "ok", "msg": "Test override"}`,
						},
					},
				},
				responseCode: 400,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"X-Foo":        []string{"initial"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Client response body",
			},
			expectedCode: 400,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"X-Foo":          []string{"initial"},
				"Content-Type":   []string{"text/plain; charset=utf-8"},
				"Content-Length": []string{strconv.Itoa(len("Client response body"))},
			},
			expectedBody: "Client response body",
		},
		{
			input: inputDataset{
				name: "replace mode",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   200,
							Headers: http.Header{
								"Content-Type": []string{"text/plain"},
							},
							Mode: "replace",
							Body: "Everything is fine",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Some error",
			},
			expectedCode: 200,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("Everything is fine"))},
			},
			expectedBody: "Everything is fine",
		},
		{
			input: inputDataset{
				name: "append mode",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   200,
							Headers: http.Header{
								"Content-Type": []string{"text/plain"},
							},
							Mode: "append",
							Body: "\nEverything is fine",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Some error",
			},
			expectedCode: 200,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("Some error\nEverything is fine"))},
			},
			expectedBody: "Some error\nEverything is fine",
		},
		{
			input: inputDataset{
				name: "prepend mode",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   200,
							Headers: http.Header{
								"Content-Type": []string{"text/plain"},
							},
							Mode: "prepend",
							Body: "Everything is fine\n",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Some error",
			},
			expectedCode: 200,
			expectedHeaders: http.Header{
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("Everything is fine\nSome error"))},
			},
			expectedBody: "Everything is fine\nSome error",
		},
		{
			input: inputDataset{
				name: "keep mode",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   400,
							Headers: http.Header{
								"X-Foo":        []string{"bar", "baz"},
								"Content-Type": []string{"text/plain"},
							},
							Mode: "keep",
							Body: "Body won't be replaced with this",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				responseBody: "Some error",
			},
			expectedCode: 400,
			expectedHeaders: http.Header{
				"X-Foo":          []string{"bar", "baz"},
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("Some error"))},
			},
			expectedBody: "Some error",
		},
		{
			input: inputDataset{
				name: "multiple rules",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{500},
							To:   400,
							Headers: http.Header{
								"X-Foo":        []string{"bar", "baz"},
								"Content-Type": []string{"text/plain"},
							},
							Mode: "replace",
							Body: "First step\n",
						},
						{
							From: []int{500},
							To:   404,
							Headers: http.Header{
								"X-Foo-2": []string{"fom"},
								"X-Foo":   []string{"far"},
							},
							Mode: "append",
							Body: "Second step\n",
						},
					},
				},
				responseCode: 500,
				responseHeaders: http.Header{
					"Server":       []string{"dummy server"},
					"Content-Type": []string{"text/plain;"},
				},
				responseBody: "Client response\n",
			},
			expectedCode: 404,
			expectedHeaders: http.Header{
				"X-Foo":          []string{"far"},
				"X-Foo-2":        []string{"fom"},
				"Server":         []string{"dummy server"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("First step\nSecond step\n"))},
			},
			expectedBody: "First step\nSecond step\n",
		},
		{
			input: inputDataset{
				name: "remove headers",
				config: changeresponse.Config{
					Overrides: []changeresponse.Override{
						{
							From: []int{200},
							To:   200,
							Headers: http.Header{
								"X-Foo": []string{"bar"},
							},
							RemoveHeaders: []string{"Server", "Transfer-Encoding", "Content-Encoding"},
							Mode:          changeresponse.ModeKeep,
						},
					},
				},
				responseCode: 200,
				responseHeaders: http.Header{
					"Server":            []string{"dummy server"},
					"Content-Type":      []string{"text/plain"},
					"Transfer-Encoding": []string{"chunked"},
					"Content-Encoding":  []string{"gzip"},
				},
				responseBody: "Client response",
			},
			expectedCode: 200,
			expectedHeaders: http.Header{
				"X-Foo":          []string{"bar"},
				"Content-Type":   []string{"text/plain"},
				"Content-Length": []string{strconv.Itoa(len("Client response"))},
			},
			expectedBody: "Client response",
		},
	}

	for _, d := range datasets {
		t.Run(d.input.name, func(t *testing.T) {
			t.Parallel()

			recorder := servePlugin(t, d.input)

			if d.expectedCode != recorder.Code {
				t.Errorf("Status code mismatch: got %d, want %d", recorder.Code, d.expectedCode)
			}

			actualBody := recorder.Body.String()
			if d.expectedBody != actualBody {
				t.Errorf("Body mismatch\nactual:   %s\nexpected: %s", actualBody, d.expectedBody)
			}

			t.Logf("Headers: %v", recorder.Header())
			if len(d.expectedHeaders) != len(recorder.Header()) {
				t.Errorf("Headers count mismatch: got %d, want %d", len(d.expectedHeaders), len(recorder.Header()))
			}

			for k := range d.expectedHeaders {
				assertHeadersEqual(t, k, recorder.Header(), d.expectedHeaders)
			}
		})
	}
}

func BenchmarkChangeResponse(t *testing.B) {
	datasets := []inputDataset{
		{
			name: "basic",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{503},
						To:   500,
						Headers: http.Header{
							"Content-Type": []string{"application/json"},
							"X-Foo":        []string{"bar", "baz"},
						},
						Body: `{"status": "ok", "msg": "Test override"}`,
					},
				},
			},
			responseCode: 503,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"X-Foo":        []string{"initial"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Client response body",
		},
		{
			name: "blank response",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   204,
						Headers: http.Header{
							"Content-Type": []string{"application/json"},
						},
						Mode: "replace",
						Body: "",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Client response body",
		},
		{
			name: "rules mismatch",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{503},
						To:   500,
						Headers: http.Header{
							"Content-Type": []string{"application/json"},
							"X-Foo":        []string{"bar", "baz"},
						},
						Body: `{"status": "ok", "msg": "Test override"}`,
					},
				},
			},
			responseCode: 400,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"X-Foo":        []string{"initial"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Client response body",
		},
		{
			name: "replace mode",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   200,
						Headers: http.Header{
							"Content-Type": []string{"text/plain"},
						},
						Mode: "replace",
						Body: "Everything is fine",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Some error",
		},
		{
			name: "append mode",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   200,
						Headers: http.Header{
							"Content-Type": []string{"text/plain"},
						},
						Mode: "append",
						Body: "\nEverything is fine",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Some error",
		},
		{
			name: "prepend mode",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   200,
						Headers: http.Header{
							"Content-Type": []string{"text/plain"},
						},
						Mode: "prepend",
						Body: "Everything is fine\n",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Some error",
		},
		{
			name: "keep mode",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   400,
						Headers: http.Header{
							"X-Foo":        []string{"bar", "baz"},
							"Content-Type": []string{"text/plain"},
						},
						Mode: "keep",
						Body: "Body won't be replaced with this",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain; charset=utf-8"},
			},
			responseBody: "Some error",
		},
		{
			name: "multiple rules",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{500},
						To:   400,
						Headers: http.Header{
							"X-Foo":        []string{"bar", "baz"},
							"Content-Type": []string{"text/plain"},
						},
						Mode: "replace",
						Body: "First step\n",
					},
					{
						From: []int{500},
						To:   404,
						Headers: http.Header{
							"X-Foo-2": []string{"fom"},
						},
						Mode: "append",
						Body: "Second step\n",
					},
				},
			},
			responseCode: 500,
			responseHeaders: http.Header{
				"Server":       []string{"dummy server"},
				"Content-Type": []string{"text/plain;"},
			},
			responseBody: "First step\nSecond step\n",
		},
		{
			name: "remove headers",
			config: changeresponse.Config{
				Overrides: []changeresponse.Override{
					{
						From: []int{200},
						To:   200,
						Headers: http.Header{
							"X-Foo": []string{"bar"},
						},
						RemoveHeaders: []string{"Server", "Transfer-Encoding", "Content-Encoding"},
						Mode:          changeresponse.ModeKeep,
					},
				},
			},
			responseCode: 200,
			responseHeaders: http.Header{
				"Server":            []string{"dummy server"},
				"Content-Type":      []string{"text/plain"},
				"Transfer-Encoding": []string{"chunked"},
				"Content-Encoding":  []string{"gzip"},
			},
			responseBody: "Client response",
		},
	}

	for _, d := range datasets {
		t.Run(d.name, func(t *testing.B) {
			servePlugin(t, d)
		})
	}
}

func servePlugin(t fatalNotifier, d inputDataset) *httptest.ResponseRecorder {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(d.responseCode)
		for k, v := range d.responseHeaders {
			for _, hv := range v {
				rw.Header().Add(k, hv)
			}
		}

		if _, err := rw.Write([]byte(d.responseBody)); err != nil {
			t.Fatal(err)
		}
	})

	handler, err := changeresponse.New(ctx, next, &d.config, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	return recorder
}

func assertHeadersEqual(t *testing.T, name string, actual http.Header, expected http.Header) {
	actualHeaders := actual.Values(name)
	sort.Strings(actualHeaders)

	expectedHeaders := expected.Values(name)
	sort.Strings(expectedHeaders)

	if !slices.Equal(actualHeaders, expectedHeaders) {
		t.Errorf(
			"%s headers not equal.\nactual:   %s\nexpected: %s",
			name,
			strings.Join(actualHeaders, ", "),
			strings.Join(expectedHeaders, ", "),
		)
	}
}

func TestCreateConfig(t *testing.T) {
	config := changeresponse.CreateConfig()
	if len(config.Overrides) != 0 {
		t.Error("Overrides should be empty by default")
	}
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	outBuf := bytes.NewBuffer([]byte{})
	changeresponse.Notify = func(msg string) {
		outBuf.WriteString(msg)
	}

	errBuf := bytes.NewBuffer([]byte{})
	changeresponse.Alert = func(msg string) {
		errBuf.WriteString(msg)
	}

	// Test 1. Check missing override rules
	config := changeresponse.CreateConfig()

	if _, err := changeresponse.New(ctx, next, config, "test-plugin"); err == nil || err.Error() != "at least one override rule is required" {
		t.Log(err)
		t.Error("Unexpected response when initializing new plugin without rules")
	}

	// Test 2. No config
	if _, err := changeresponse.New(ctx, next, nil, "test-plugin"); err == nil || err.Error() != "config must be defined" {
		t.Log(err)
		t.Error("Unexpected response when initializing new plugin without config")
	}

	// Test 3. Debug message with successful init
	config = &changeresponse.Config{
		Overrides: []changeresponse.Override{{
			From: []int{200},
			To:   200,
		}},
		Debug: true,
	}

	outBuf.Reset()
	if _, err := changeresponse.New(ctx, next, config, "test-plugin"); err != nil {
		t.Error("Unexpected error: " + err.Error())
	}

	if !strings.Contains(outBuf.String(), "defined config test-plugin:") {
		t.Errorf(
			"Unexpected notification\nactual: %s\nexpected: %s",
			outBuf.String(),
			"defined config test-plugin:",
		)
	}
}
