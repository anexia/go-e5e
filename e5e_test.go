package e5e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

type SumData struct {
	A int `json:"a"`
	B int `json:"b"`
}

type SumEvent struct {
	Event
	Data SumData `json:"data"`
}

type unexportedEvent struct {
	Event
	Data SumData `json:"data"`
}

type unexportedResult struct {
	Result
}

type entrypoints struct{}

var defaultPayload = map[string]interface{}{
	"event": map[string]interface{}{
		"params": map[string][]string{
			"test-param": {"a", "b"},
		},
		"data": map[string]interface{}{
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
		"type":  "object",
	},
}

func TestStartSimpleEntrypoint(t *testing.T) {
	assertExecutionNormal(t, "SimpleEntrypoint", defaultPayload, expectedData{result: "{\"result\":null}"})
}

func TestStartInvalidSimpleEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while preparing entrypoint: entrypoint InvalidSimpleEntrypoint not found",
	}
	assertExecutionNormal(t, "InvalidSimpleEntrypoint", defaultPayload, expected)
}

func TestStartSumEntrypoint(t *testing.T) {
	expected := expectedData{
		result: "{\"result\":{\"data\":5}}",
	}
	assertExecutionNormal(t, "SumEntrypoint", defaultPayload, expected)
}

func TestStartSumEntrypointKeepalive(t *testing.T) {
	expected := expectedData{
		result: "{\"result\":{\"data\":5}}",
	}

	payloadBytes, err := json.Marshal(defaultPayload)
	if err != nil {
		t.Fatalf("Preparing payload JSON failed: %v", err)
	}
	payloadBytes = append(payloadBytes, '\n')

	commands := []byte("ping\nping\n")
	commands = append(commands, payloadBytes...)
	commands = append(commands, []byte("ping\n")...)
	commands = append(commands, payloadBytes...)

	stdoutValue, stderrValue := assertExecution(t, "SumEntrypoint", commands, true, expected)

	expectedOutputs := []string{"pong", "pong", "\x00\x00\x00\x00\x00{\"result\":{\"data\":5}}", "pong", "\x00\x00\x00\x00\x00{\"result\":{\"data\":5}}", ""}
	expectedStdout := strings.Join(expectedOutputs, "\x00\x00\x00\x00\x00\x00")

	expectedStderr := strings.Repeat("\x00\x00\x00\x00\x00\x00", 5)

	if stdoutValue != expectedStdout {
		t.Errorf("Invalid stdout: %s", stdoutValue)
	}

	if stderrValue != expectedStderr {
		t.Errorf("Invalid stderr: %s", stderrValue)
	}

}

func TestStartSumNoPtrEntrypoint(t *testing.T) {
	expected := expectedData{
		result: "{\"result\":{\"data\":5}}",
	}
	assertExecutionNormal(t, "SumEntrypointNoPtr", defaultPayload, expected)
}

func TestStartEventContextEntrypoint(t *testing.T) {
	expected := expectedData{
		result: "{\"result\":{\"data\":{\"context\":{\"date\":\"2022-08-04T14:15:53.885414\",\"type\":\"object\"},\"event\":{\"params\":{\"test-param\":[\"a\",\"b\"]},\"request_headers\":{\"test-header\":\"test-header-value\"},\"type\":\"object\",\"data\":{\"a\":2,\"b\":3}}}}}",
	}
	assertExecutionNormal(t, "EventContextEntrypoint", defaultPayload, expected)
}

func TestStartPrintStdOutEntrypoint(t *testing.T) {
	expected := expectedData{
		stdout: "print",
		stderr: "error print",
		result: "{\"result\":null}",
	}
	assertExecutionNormal(t, "PrintStdOutErrEntrypoint", defaultPayload, expected)
}

func TestStartErrorEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while executing entrypoint: entrypoint returned error error",
	}
	assertExecutionNormal(t, "ErrorEntrypoint", defaultPayload, expected)
}

func TestStartInvalidParametersEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while preparing entrypoint: invalid number of entrypoint parameters on InvalidParametersEntrypoint",
	}
	assertExecutionNormal(t, "InvalidParametersEntrypoint", defaultPayload, expected)
}

func TestStartInvalidParameterTypesEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while executing entrypoint: error while parsing json json: cannot unmarshal object into Go struct field .context of type uint",
	}
	assertExecutionNormal(t, "InvalidParameterTypesEntrypoint", defaultPayload, expected)
}

func TestStartInvalidResultEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while preparing entrypoint: invalid number of entrypoint return values on InvalidResultEntrypoint",
	}
	assertExecutionNormal(t, "InvalidResultEntrypoint", defaultPayload, expected)
}

func TestStartInvalidResultValueEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while processing function response: json: unsupported value: +Inf",
	}
	assertExecutionNormal(t, "InvalidResultValueEntrypoint", defaultPayload, expected)
}

func TestStartInvalidErrorResultValueEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while executing entrypoint: invalid error return value",
	}
	assertExecutionNormal(t, "InvalidErrorResultValueEntrypoint", defaultPayload, expected)
}

func TestStartInvalidUnexportedEntrypoint(t *testing.T) {
	expected := expectedData{
		exitCode: -1,
		err:      "go-e5e: error while preparing entrypoint: entrypoint invalidUnexportedEntrypoint not found",
	}
	assertExecutionNormal(t, "invalidUnexportedEntrypoint", defaultPayload, expected)
}

type expectedData struct {
	stderr   string
	stdout   string
	result   string
	exitCode int
	err      string
}

func assertExecutionNormal(t *testing.T, entrypoint string, payload map[string]interface{}, expected expectedData) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Preparing payload JSON failed: %v", err)
	}
	payloadBytes = append(payloadBytes, '\n')
	stdoutValue, stderrValue := assertExecution(t, entrypoint, payloadBytes, false, expected)

	stdoutTerminator := "\x00\x00\x00\x00\x00"

	if expected.err != "" {
		stdoutTerminator = ""
	}

	expectedStdout := fmt.Sprintf("%s%s%s", expected.stdout, stdoutTerminator, expected.result)
	if stdoutValue != expectedStdout {
		t.Errorf("Invalid stdout: %s", stdoutValue)
	}

	if stderrValue != expected.stderr {
		t.Errorf("Invalid stderr: %s", stderrValue)
	}
}

func assertExecution(t *testing.T, entrypoint string, payload []byte, keepalive bool, expected expectedData) (string, string) {
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()
	stdinReader, stdinWriter, _ := os.Pipe()

	origStdout := os.Stdout
	origStderr := os.Stderr
	origStdin := os.Stdin
	origArgs := os.Args

	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		os.Stdin = origStdin
		os.Args = origArgs
	}()

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	os.Stdin = stdinReader

	stdoutSignal := make(chan string)
	stderrSignal := make(chan string)

	go func() {
		_, _ = stdinWriter.Write(payload)
	}()

	go func() {
		buffer := &bytes.Buffer{}
		_, _ = io.Copy(buffer, stdoutReader)
		stdoutSignal <- buffer.String()
	}()

	go func() {
		buffer := &bytes.Buffer{}
		_, _ = io.Copy(buffer, stderrReader)
		stderrSignal <- buffer.String()
	}()

	keepaliveString := "0"
	if keepalive {
		keepaliveString = "1"
	}
	os.Args = []string{os.Args[0], entrypoint, "\x00\x00\x00\x00\x00", keepaliveString, "\x00\x00\x00\x00\x00\x00"}

	execSignal := make(chan interface{})
	go func() {
		exitCode, err := start(&entrypoints{})

		if exitCode != expected.exitCode {
			t.Errorf("Invalid exit code %d", exitCode)
		}

		// Error was expected but none was returned
		if err == nil && expected.err != "" {
			t.Error("Execution did not return error")
		}

		if err != nil && err.Error() != expected.err {
			t.Errorf("Execution returned invalid error %v", err)
		}

		close(execSignal)
	}()

	if !keepalive {
		<-execSignal
	} else {
		time.Sleep(1 * time.Second)
	}

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	stdoutValue := <-stdoutSignal
	stderrValue := <-stderrSignal

	return stdoutValue, stderrValue
}

func (f *entrypoints) SimpleEntrypoint(event Event, context Context) (*Result, error) {
	return nil, nil
}

func (f *entrypoints) SumEntrypoint(event SumEvent, context Context) (*Result, error) {
	return &Result{
		Data: event.Data.A + event.Data.B,
	}, nil
}

func (f *entrypoints) SumEntrypointNoPtr(event SumEvent, context Context) (Result, error) {
	return Result{
		Data: event.Data.A + event.Data.B,
	}, nil
}

func (f *entrypoints) EventContextEntrypoint(event SumEvent, context Context) (*Result, error) {
	return &Result{
		Data: map[string]interface{}{
			"event":   event,
			"context": context,
		},
	}, nil
}

func (f *entrypoints) PrintStdOutErrEntrypoint(event Event, context Context) (*Result, error) {
	fmt.Print("print")
	_, _ = fmt.Fprint(os.Stderr, "error print")
	return nil, nil
}

func (f *entrypoints) ErrorEntrypoint(event Event, context Context) (*Result, error) {
	return nil, fmt.Errorf("error")
}

func (f *entrypoints) InvalidParametersEntrypoint() (*Result, error) {
	return nil, nil
}

func (f *entrypoints) InvalidParameterTypesEntrypoint(event uint, context uint) (*Result, error) {
	return nil, nil
}

func (f *entrypoints) InvalidResultEntrypoint(event Event, context Context) {
	return
}

func (f *entrypoints) InvalidResultValueEntrypoint(event Event, context Context) (*Result, error) {
	return &Result{
		Data: math.Inf(1),
	}, nil
}

func (f *entrypoints) InvalidErrorResultValueEntrypoint(event Event, context Context) (*Result, int) {
	return nil, 1
}

func (f *entrypoints) invalidUnexportedEntrypoint(event Event, context Context) (*Result, error) {
	return nil, nil
}
