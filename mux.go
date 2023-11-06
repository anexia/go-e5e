package e5e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
)

// mux defines a container for entrypoints and routes the requests for the given entrypoint
// to the dedicated handlers.
type mux struct {
	handlers    map[string]HandlerFactory
	stdinReader *bufio.Scanner

	lock sync.Mutex
}

var globalMux = &mux{handlers: make(map[string]HandlerFactory)}

func init() {
	if len(os.Args) == 2 && os.Args[1] == "metadata" {
		writeMetadata()
		os.Exit(0)
	}
}

// Start starts the global mux.
//
// The runtime arguments are read from os.Args and based on that,
// the entrypoint which was added via [AddHandlerFunc] before that gets called.
//
// Every error that is happening will cause a panic.
func Start(ctx context.Context) {
	args, err := parseArguments(os.Args)
	if err != nil {
		panic(err)
	}

	err = globalMux.Start(ctx, args)
	if err != nil {
		panic(err)
	}
}

// AddHandlerFunc adds the handler for the given entrypoint to the global handler.
// It panics if the entrypoint was already registered.
func AddHandlerFunc[T, TContext Data](entrypoint string, fn func(context.Context, Request[T, TContext]) (*Result, error)) {
	if err := addHandlerSafely(globalMux, entrypoint, createHandlerFunc[T, TContext](fn)); err != nil {
		panic(err)
	}
}

// addHandlerSafely adds the handler for the given entrypoint to the mux.
// If there's an error, usually by registering the same entrypoint twice, an error is returned.
func addHandlerSafely[T, TContext Data](m *mux, entrypoint string, handler Handler[T, TContext]) error {
	_, exists := m.handlers[entrypoint]
	if exists {
		return fmt.Errorf("entrypoint %q is already registered on this mux", entrypoint)
	}

	m.handlers[entrypoint] = &typedHandlerFactory[T, TContext]{h: handler}
	return nil
}

// parseArguments takes a list of arguments, usually [os.Args] and parses them into a valid [options].
//
// # Rules
//
// The following rules apply:
//
//   - args must contain either two or five elements.
//   - If it contains two elements, the second element must be `metadata` in order for [options.WriteMetadata] to be true.
//
// # Argument order
//
// Otherwise, the arguments need to be in the following order:
//
//  1. The name of the binary.
//  2. The entrypoint that should be called, used for determining the [Handler].
//  3. The standard output termination sequence, which is written *before* writing the serialized JSON response.
//  4. Whether the daemon should be kept alive, determined by a 0 (false) or a 1 (true).
//  5. The daemon execution termination sequence, which is written *after* writing the *successful* JSON response.
func parseArguments(args []string) (options, error) {
	// Check number of arguments:
	// binary name, entrypoint, os.Stdout termination, keepalive enabled, daemon execution termination
	if argCount := len(args); argCount != 5 {
		return options{}, fmt.Errorf("invalid number of process arguments: %d", argCount)
	}

	res := options{
		Entrypoint:                         args[1],
		StdoutExecutionSequence:            strings.ReplaceAll(args[2], "\\0", "\x00"),
		KeepAlive:                          args[3] == "1",
		DaemonExecutionTerminationSequence: strings.ReplaceAll(args[4], "\\0", "\x00"),
	}
	return res, nil
}

// Start starts the mux and returns potential runtime errors.
//
// The startup arguments are read from [os.Args], however it's possible to set custom startup options
// by setting the options on this mux.
//
// If [options.KeepAlive] is true, the goroutine is blocked and can be cancelled
// via the context. It also listens for incoming [syscall.SIGINT] signals and stops gracefully.
//
// # Example usage
//
//	mux := e5e.NewMux()
//	log.Fatalf("mux error: %v", mux.Start(context.Background()))
func (m *mux) Start(ctx context.Context, opts options) error {
	if _, hasEntrypoint := m.handlers[opts.Entrypoint]; !hasEntrypoint {
		return InvalidEntrypointError{opts.Entrypoint}
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Lock the stdin until the supportive goroutine doesn't need it anymore.
	m.lock.Lock()
	m.stdinReader = bufio.NewScanner(os.Stdin)
	m.stdinReader.Buffer([]byte{}, 1024*1024*1024) // 1 GiB

	// Read the lines in the background and cancel the reading with the given context.
	lineChan := make(chan []byte)
	go func(ctx context.Context) {
		defer m.lock.Unlock()
		defer close(lineChan)
	loop:
		for m.stdinReader.Scan() {
			select {
			case <-ctx.Done():
				break loop
			default:
				b := m.stdinReader.Bytes()
				if len(b) == 0 {
					continue
				}

				lineChan <- b
			}
		}

		if err := m.stdinReader.Err(); err != nil {
			panic(fmt.Sprintf("go-e5e: reading from stdin failed: %v", err))
		}
	}(ctx)

	for line := range lineChan {
		response, err := m.execute(ctx, line, opts)
		if err != nil {
			return fmt.Errorf("go-e5e: executing handler: %w", err)
		}

		_, _ = fmt.Fprint(os.Stdout, opts.StdoutExecutionSequence)
		_, _ = fmt.Fprint(os.Stdout, response)

		// In case this is a single execution exit the loop
		if !opts.KeepAlive {
			break
		}

		// Print execution termination signals
		_, _ = fmt.Fprint(os.Stdout, opts.DaemonExecutionTerminationSequence)
		_, _ = fmt.Fprint(os.Stderr, opts.DaemonExecutionTerminationSequence)
	}

	return nil
}

// execute reads a line from the input, parses it and returns the response that should be written.
func (m *mux) execute(ctx context.Context, payload []byte, opts options) (string, error) {
	if string(payload) == "ping" && opts.KeepAlive {
		return "pong", nil
	}

	factory := m.handlers[opts.Entrypoint]
	res, err := factory.Execute(ctx, payload)
	if err != nil {
		return "", fmt.Errorf("handler execution: %w", err)
	}

	wrapped := struct {
		Result *Result `json:"result"`
	}{Result: res}

	resp, err := json.Marshal(wrapped)
	if err != nil {
		return "", fmt.Errorf("marshalling response: %w", err)
	}

	return string(resp), nil
}

// write the metadata that's used by e5e for the dashboard to [os.Stdout]
func writeMetadata() {
	type metadata struct {
		LibraryVersion string   `json:"library_version"`
		Runtime        string   `json:"runtime"`
		RuntimeVersion string   `json:"runtime_version"`
		Features       []string `json:"features"`
	}

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
}
