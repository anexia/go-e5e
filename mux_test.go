package e5e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"go.anx.io/e5e/v2"
)

type IntegrationTestPayload struct {
	A int `json:"a"`
	B int `json:"b"`
}

type IntegrationTestContext struct {
	AuthKey string `json:"Auth-Key"`
}

var (
	stdoutTerminationSequence = strings.Repeat("\x00", 5)
	daemonTerminationSequence = strings.Repeat("\x00", 6)
)

func buildOptions(entrypoint string) []string {
	return []string{
		"test-binary",
		entrypoint,
		stdoutTerminationSequence,
		"0",
		daemonTerminationSequence,
	}
}

// we use a simplified type here in order to make the tests less cluttered
type testHandlerFunc func(*testing.T, e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error)

var (
	defaultPayload, _ = json.Marshal(map[string]interface{}{
		"event": map[string]interface{}{
			"params": map[string][]string{
				"test-param": {"a", "b"},
			},
			"data": map[string]int{
				"a": 2,
				"b": 3,
			},
			"request_headers": map[string]string{
				"test-header": "test-header-value",
			},
			"type": "object",
		},
		"context": map[string]interface{}{
			"async": false,
			"date":  "2022-08-04T14:15:53.885414",
			"type":  "go-library-test",
			"data": map[string]string{
				"Auth-Key": "my-auth-key",
			},
		},
	})
	expectedRequest = e5e.Request[IntegrationTestPayload, IntegrationTestContext]{
		Event: e5e.Event[IntegrationTestPayload]{
			Params: map[string][]string{
				"test-param": {"a", "b"},
			},
			RequestHeaders: map[string]string{
				"test-header": "test-header-value",
			},
			Type: e5e.EventDataTypeObject,
			Data: IntegrationTestPayload{A: 2, B: 3},
		},
		Context: e5e.Context[IntegrationTestContext]{
			Async: false,
			Date:  "2022-08-04T14:15:53.885414",
			Type:  "go-library-test",
			Data: IntegrationTestContext{
				AuthKey: "my-auth-key",
			},
		},
	}
	defaultHandler testHandlerFunc = func(*testing.T, e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
		return &e5e.Result{}, nil
	}
)

func invokeE5E(t *testing.T, stdin string) (stdout, stderr string) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	stdio := redirectStdio(t, stdin)
	os.Args = buildOptions(t.Name())

	e5e.Start(ctx)

	stdout, stderr = stdio.ReadAndRestore()
	return
}

func TestResponse(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		handler testHandlerFunc
		result  string
		stdout  string
		stderr  string
	}{
		{
			name: "empty result",
			handler: func(*testing.T, e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return &e5e.Result{Data: nil}, nil
			},
			result: `{"result":{"data":null}}`,
		},
		{
			name: "nil result",
			handler: func(*testing.T, e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return &e5e.Result{Status: 200}, nil
			},
			result: `{"result":{"status":200,"data":null}}`,
		},
		{
			name:   "print stdout",
			stdout: "print",
			stderr: "error print",
			handler: func(*testing.T, e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				fmt.Print("print")
				_, _ = fmt.Fprint(os.Stderr, "error print")
				return nil, nil
			},
			result: `{"result":null}`,
		},
		{
			name: "sum",
			handler: func(t *testing.T, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return &e5e.Result{Data: r.Data().A + r.Data().B}, nil
			},
			result: `{"result":{"data":5}}`,
		},
		{
			name: "request contains all keys and values",
			handler: func(t *testing.T, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				DeepEqual(t, r, expectedRequest, "request does not match")
				return &e5e.Result{}, nil
			},
			result: `{"result":{"data":null}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.stdin) == 0 {
				tt.stdin = string(defaultPayload)
			}
			if tt.handler == nil {
				tt.handler = defaultHandler
			}

			e5e.AddHandlerFunc(t.Name(), func(ctx context.Context, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return tt.handler(t, r)
			})

			stdout, stderr := invokeE5E(t, tt.stdin)
			expectedStdout := fmt.Sprintf("%s%s%s", tt.stdout, stdoutTerminationSequence, tt.result)

			Equal(t, expectedStdout, stdout, "stdout does not match")
			Equal(t, tt.stderr, stderr, "stderr does not match")
		})
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name       string
		entrypoint string
		handler    testHandlerFunc
		error      error
	}{
		{
			name:       "invalid entrypoint",
			entrypoint: "does_not_exist",
			error:      &e5e.InvalidEntrypointError{Entrypoint: "does_not_exist"},
		},
		{
			name: "handler returns error",
			handler: func(t *testing.T, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return nil, errors.New("error")
			},
		},
		{
			name: "invalid result (infinity)",
			handler: func(t *testing.T, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return &e5e.Result{Data: math.Inf(0)}, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler == nil {
				tt.handler = defaultHandler
			}

			e5e.AddHandlerFunc(t.Name(), func(ctx context.Context, r e5e.Request[IntegrationTestPayload, IntegrationTestContext]) (*e5e.Result, error) {
				return tt.handler(t, r)
			})

			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected panic, got none")
				}

				err, _ := r.(error)
				if tt.error != nil {
					Equal(t, tt.error.Error(), err.Error(), "errors do not match")
				}
			}()

			entrypoint := t.Name()
			if tt.entrypoint != "" {
				entrypoint = tt.entrypoint
			}
			os.Args = buildOptions(entrypoint)

			redirectStdio(t, string(defaultPayload))
			e5e.Start(context.Background())
		})
	}
}

func TestKeepAlive(t *testing.T) {
	var payload strings.Builder
	payload.WriteString("ping")
	payload.WriteRune('\n')
	payload.WriteString("ping")
	payload.WriteRune('\n')
	payload.Write(defaultPayload)
	payload.WriteRune('\n')
	payload.WriteString("ping")
	payload.WriteRune('\n')
	payload.Write(defaultPayload)
	payload.WriteRune('\n')

	e5e.AddHandlerFunc(t.Name(), func(ctx context.Context, r e5e.Request[IntegrationTestPayload, any]) (*e5e.Result, error) {
		return &e5e.Result{Data: r.Data().A + r.Data().B}, nil
	})

	stdio := redirectStdio(t, payload.String())
	os.Args = []string{
		"test-binary",
		t.Name(),
		stdoutTerminationSequence,
		"1",
		daemonTerminationSequence,
	}

	e5e.Start(context.Background())
	stdout, stderr := stdio.ReadAndRestore()

	expectedOutputs := []string{
		"pong",
		"pong",
		`{"result":{"data":5}}`,
		"pong",
		`{"result":{"data":5}}`,
	}
	var expectedStdout strings.Builder
	for _, v := range expectedOutputs {
		expectedStdout.WriteString(stdoutTerminationSequence)
		expectedStdout.WriteString(v)
		expectedStdout.WriteString(daemonTerminationSequence)
	}

	expectedStderr := strings.Repeat(daemonTerminationSequence, len(expectedOutputs))

	Equal(t, expectedStdout.String(), stdout, "stdout does not match")
	Equal(t, expectedStderr, stderr, "stderr does not match")
}

func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel it immediately

	stdio := redirectStdio(t, string(defaultPayload))

	e5e.AddHandlerFunc(t.Name(), func(ctx context.Context, r e5e.Request[IntegrationTestPayload, any]) (*e5e.Result, error) {
		return &e5e.Result{
			Data: r.Data().A + r.Data().B,
		}, nil
	})
	e5e.Start(ctx)

	stdout, stderr := stdio.ReadAndRestore()

	// We shouldn't even have any content here.
	Equal(t, "", stdout, "stdout does not match")
	Equal(t, "", stderr, "stderr does not match")
}

func Equal[T comparable](t *testing.T, expected, actual T, message string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s:\n\tgot:\t%v\n\twanted:\t%v", message, actual, expected)
	}
}

func DeepEqual(t *testing.T, expected interface{}, actual interface{}, message string) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s:\n\tgot:\t%+v\n\twanted:\t%+v", message, actual, expected)
	}
}
