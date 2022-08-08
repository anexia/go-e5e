package e5e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
)

// Init to enforce handling of the metadata command used by e5e to determine the supported feature set.
func init() {
	type metadata struct {
		LibraryVersion string   `json:"library_version"`
		Runtime        string   `json:"runtime"`
		RuntimeVersion string   `json:"runtime_version"`
		Features       []string `json:"features"`
	}

	if len(os.Args) == 2 && os.Args[1] == "metadata" {
		metadataInstance := metadata{
			LibraryVersion: LibraryVersion,
			Runtime:        "Go",
			RuntimeVersion: runtime.Version(),
			Features:       []string{"keepalive"},
		}
		metadataBytes, err := json.Marshal(metadataInstance)
		if err != nil {
			panic(fmt.Errorf("go-e5e: metadata generation failed: %w", err))
		}
		_, _ = os.Stdout.Write(metadataBytes)
		os.Exit(0)
	}
}

// Event represents the passed `event` object of an e5e function. Contains all fields but `data`, as the user code is
// expected to encapsulate this struct within its own struct containing the `data` definition.
type Event struct {
	Params         map[string][]string `json:"params,omitempty"`
	RequestHeaders map[string]string   `json:"request_headers,omitempty"`
	Type           string              `json:"type,omitempty"`
}

// Context represents the passed `context` object of an e5e function. Contains all fields but `data`, as the user code
// is expected to encapsulate this struct within its own struct containing the `data` definition when necessary.
type Context struct {
	Async bool   `json:"async,omitempty"`
	Date  string `json:"date,omitempty"`
	Type  string `json:"type,omitempty"`
}

// Result represents the function result value passed back to E5E.
type Result struct {
	Status          int               `json:"status,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	Data            interface{}       `json:"data,omitempty"`
	Type            string            `json:"type,omitempty"`
}

// Struct for the internal e5e response representation.
type response struct {
	Result interface{} `json:"result"`
}

// Represents metadata for an entrypoint execution.
type execution struct {
	stdoutTermination          string
	keepalive                  bool
	daemonExecutionTermination string
	entrypoint                 reflect.Value
	payloadType                reflect.Type
	stdinReader                *bufio.Reader
}

// LibraryVersion represents the implemented custom binary interface version.
//
//goland:noinspection GoUnusedConst
const LibraryVersion = "1.2.0"

// Start takes the struct containing the available entrypoint methods and handles the invocation of the
// entrypoint as well as the communication with the e5e platform itself. Control will not be handed back after
// function execution. In case of an error a Go panic will be raised otherwise an os.Exit(0) occurs.
//
// Rules:
//   - Entrypoint functions must take 2 input parameters (Event and Context). Both types may be encapsulated within
//     an user defined struct type.
//   - Entrypoint functions must return 2 values (Result and error). Type encapsulation is also allowed here.
//   - The input parameters as well as the return values must be compatible with "encoding/json" standard library.
//
//goland:noinspection GoUnusedExportedFunction
func Start(entrypoints interface{}) {
	exitCode, err := start(entrypoints)
	if err != nil {
		panic(err)
	}
	os.Exit(exitCode)
}

// Internal implementation of the Start logic which returns an exit code and any available error object.
func start(entrypoints interface{}) (int, error) {
	executionInstance := execution{}

	// Init execution instance
	if err := executionInstance.init(entrypoints); err != nil {
		return -1, fmt.Errorf("go-e5e: %w", err)
	}

	// Start execution loop
	for {
		if err := executionInstance.execute(); err != nil {
			return -1, fmt.Errorf("go-e5e: %w", err)
		}

		// In case this is a single execution exit the loop
		if !executionInstance.keepalive {
			break
		}

		// Print execution termination signals
		_, _ = fmt.Fprint(os.Stdout, executionInstance.daemonExecutionTermination)
		_, _ = fmt.Fprint(os.Stderr, executionInstance.daemonExecutionTermination)
	}

	return 0, nil
}

// Populates the execution struct with values given as command line arguments and fetches the required entrypoint. Will
// return an error on any invalid condition.
func (e *execution) init(entrypoints interface{}) error {
	// Check number of arguments:
	// binary name, entrypoint, os.Stdout termination, keepalive enabled, daemon execution termination
	if argCount := len(os.Args); argCount != 5 {
		return fmt.Errorf("invalid number of process arguments: %d", argCount)
	}

	e.stdoutTermination = strings.ReplaceAll(os.Args[2], "\\0", "\x00")
	e.keepalive = os.Args[3] == "1"
	e.daemonExecutionTermination = strings.ReplaceAll(os.Args[4], "\\0", "\x00")
	e.stdinReader = bufio.NewReader(os.Stdin)

	var err error
	e.entrypoint, e.payloadType, err = getEntrypoint(entrypoints, os.Args[1])
	if err != nil {
		return fmt.Errorf("error while preparing entrypoint: %w", err)
	}

	return nil
}

// Tries to execute the preconfigured entrypoint or responds to a `ping` command in keepalive mode. Will return an error
// on any invalid condition.
func (e *execution) execute() error {
	payloadString, err := e.readStdin()
	if err != nil {
		return fmt.Errorf("error while reading os.Stdin: %w", err)
	}

	if e.keepalive && payloadString == "ping" {
		_, _ = fmt.Fprint(os.Stdout, "pong")
		return nil
	}

	result, err := e.callEntrypoint(payloadString)
	if err != nil {
		return fmt.Errorf("error while executing entrypoint: %w", err)
	}

	responseBytes, err := json.Marshal(response{Result: result})
	if err != nil {
		return fmt.Errorf("error while processing function response: %w", err)
	}

	_, _ = fmt.Fprint(os.Stdout, e.stdoutTermination)
	_, _ = os.Stdout.Write(responseBytes)

	return nil
}

// Parses the payload for entrypoint execution, execute the entrypoint and return the validated result. Will return an
// error on any invalid condition.
func (e *execution) callEntrypoint(payloadString string) (interface{}, error) {
	payload := reflect.New(e.payloadType)
	payloadInterface := payload.Interface()
	payloadElem := payload.Elem()

	if err := json.Unmarshal([]byte(payloadString), &payloadInterface); err != nil {
		return nil, fmt.Errorf("error while parsing json %w", err)
	}

	results := e.entrypoint.Call([]reflect.Value{
		payloadElem.Field(0),
		payloadElem.Field(1),
	})

	if results[1].Kind() != reflect.Interface || !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, fmt.Errorf("entrypoint returned error %w", err)
		} else {
			return nil, fmt.Errorf("invalid error return value")
		}
	}

	return results[0].Interface(), nil
}

// Reads a line terminated by `\n` from os.Stdin and returns it together with any occurring error.
func (e *execution) readStdin() (string, error) {
	line, err := e.stdinReader.ReadString('\n')
	if err != nil {
		return line, err
	}
	return strings.TrimRight(line, "\n"), nil
}

// Fetches the given endpoint name from the endpoints struct. Will then try to create a payload struct type based on
// the used parameter types. Will return an error on any invalid condition.
func getEntrypoint(entrypoints interface{}, name string) (entrypoint reflect.Value, payload reflect.Type, err error) {
	// The following reflect calls have a tendency to panic if anything does not work out as we expect.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while trying to fetch entrypoint %v", r)
		}
	}()

	// Fetch entrypoint
	entrypoint = reflect.ValueOf(entrypoints).MethodByName(name)

	// Validate entrypoint
	if !entrypoint.IsValid() {
		err = fmt.Errorf("entrypoint %s not found", name)
		return
	}
	if entrypoint.Type().NumIn() != 2 {
		err = fmt.Errorf("invalid number of entrypoint parameters on %s", name)
		return
	}
	if entrypoint.Type().NumOut() != 2 {
		err = fmt.Errorf("invalid number of entrypoint return values on %s", name)
		return
	}

	// As we are now as sure as we can get that the method signature is the expected one, we receive the type
	// information of the first and the second parameter, and create a new references to instances of those types.
	eventType := entrypoint.Type().In(0)
	contextType := entrypoint.Type().In(1)
	payload = reflect.StructOf([]reflect.StructField{
		{
			Name: "Event",
			Type: eventType,
			Tag:  `json:"event"`,
		},
		{
			Name: "Context",
			Type: contextType,
			Tag:  `json:"context"`,
		},
	})
	return
}
