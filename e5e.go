// Package e5e helps Anexia customers who want to use the e5e Functions-as-a-Service (FaaS)
// offering in the [Anexia Engine].
//
// It provides a simple runtime which properly handles the input from the runtime.
//
// [Anexia Engine]: https://engine.anexia-it.com/docs/en/module/e5e/
package e5e // import "go.anx.io/e5e/v2"

// EventDataType tells more information about the type of the data inside an [Event].
type EventDataType string

const (
	// EventDataTypeText tells us that the data is a plain string.
	//
	// Equivalent to the `text/plain` content type.
	EventDataTypeText EventDataType = "text"

	// EventDataTypeObject tells us that the data is a JSON object.
	//
	// The data is either a primitive data type (bool, int) or contains structured
	// data like a struct or a map.
	//
	// Equivalent to the `application/*` content type.
	EventDataTypeObject EventDataType = "object"

	// EventDataTypeBinary tells us that the data is a base64 encoded string.
	//
	// The data contains a binary object representation of the response body.
	EventDataTypeBinary EventDataType = "binary"

	// EventDataTypeMixed tells us about mixed data.
	//
	// The data contains a `map[string][]any` where each key represents a field
	// name submitted by the client. Since a field name may occur multiple times
	// within one request, the values of a field are always given as a list.
	// Each value might be of a primitive data type such as string, int, bool, nil
	// or it might be a binary object representation.
	//
	// Equivalent to the `multipart/form-data` content type.
	EventDataTypeMixed EventDataType = "mixed"
)

// The Data constraint is used to constrain the incoming data for [Event] and [Context].
// It's equal to any, because there's not a good constraint which would be equivalent to `Serializable`.
type Data any

// Event contains all sorts of information about the event that triggered the
// execution, such as GET parameters, request headers, input data and the type of the input data.
type Event[T Data] struct {
	// Params contains the GET parameters of the request.
	// As GET parameters can occur multiple times within a single request,
	// the values are given as a list.
	Params map[string][]string `json:"params,omitempty"`

	// Contains the HTTP headers that were sent with this request.
	RequestHeaders map[string]string `json:"request_headers,omitempty"`

	// The type of the data in [Data].
	Type EventDataType `json:"type,omitempty"`

	// The data that's submitted with the request.
	Data T `json:"data,omitempty"`
}

// Context represents the passed `context` object of an e5e function.
type Context[T Data] struct {
	// Set to true if the event was triggered in an asynchronous way,
	// meaning that the event trigger does not wait for the return of the
	// function execution.
	Async bool `json:"async,omitempty"`

	// The time the event was triggered.
	Date string `json:"date,omitempty"`

	// The kind of trigger that triggered the execution.
	// Fallback is `generic`, if the trigger is unknown.
	Type string `json:"type,omitempty"`

	// Additional data about the context.
	Data T `json:"data,omitempty"`
}

// Request contains the whole request information.
type Request[T, TContext Data] struct {
	Context Context[TContext] `json:"context"`
	Event   Event[T]          `json:"event"`
}

// Data provides a shortcut to [r.Event.Data].
func (r Request[T, TContext]) Data() T { return r.Event.Data }

// Result represents the function result value passed back to E5E.
type Result struct {
	Status          int               `json:"status,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	Data            any               `json:"data"`
	Type            ResultDataType    `json:"type,omitempty"`
}

// ResultDataType tells more information about the type of the data inside a [Result].
type ResultDataType string

const (
	// ResultDataTypeText tells E5E that the data is a plain string.
	//
	// Equivalent to the `text/plain` content type.
	ResultDataTypeText ResultDataType = "text"

	// ResultDataTypeObject tells E5E that the data is a JSON object.
	//
	// The data is either a primitive data type (bool, int) or contains structured
	// data like a struct or a map.
	//
	// Equivalent to the `application/*` content type.
	ResultDataTypeObject ResultDataType = "object"

	// ResultDataTypeBinary tells E5E that the data is a base64 encoded string.
	//
	// The data contains a binary object representation of the response body.
	ResultDataTypeBinary ResultDataType = "binary"
)

// LibraryVersion represents the implemented custom binary interface version.
//
//goland:noinspection GoUnusedConst
const LibraryVersion = "2.0.0"

// options contains all the runtime options that determine the behaviour of the [mux].
// It is usually read at runtime using [parseArguments], but can be overridden for testing.
type options struct {
	// The name of the entrypoint that is executed on incoming events.
	Entrypoint string

	// The termination sequence that should be written on shutdown.
	DaemonExecutionTerminationSequence string

	// The execution sequence that separates generic output on [os.Stdout] from the encoded responses.
	StdoutExecutionSequence string

	// If set to true, the application is kept alive after the first execution and responds to ping events.
	KeepAlive bool
}
