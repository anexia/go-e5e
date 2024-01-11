package e5e

import (
	"context"
	"encoding/json"
	"fmt"
)

// A Handler responds to a request.
//
// Handle should write the response data with optional metadata to the given result.
// Returning signals that the request is finished.
//
// Except for reading the body, handlers should not modify the provided [Event].
type Handler[T, TContext Data] interface {
	// Handle receives an event during runtime.
	//
	// If the execution returns an error, the request is considered failed.
	// In all other cases, including both values being nil, the request is successful.
	// Although the provided context is not used at this moment, it is kept for forward compatibility for future
	// enhancements of the E5E runtime.
	Handle(context.Context, Request[T, TContext]) (*Result, error)
}

// HandlerFactory provides an interface to a struct that wraps a job to be done
// combined with a function that can execute it. Its main purpose is to
// wrap a struct that contains generic types (like a Handler[T, TContext] that needs to be
// invoked with a Request[T, TContext]) in such a way as to make it non-generic so that it can
// be used in other non-generic code like the global mux.
type HandlerFactory interface {
	// Execute the handler with payload, which should be a deserializable JSON object.
	// Any errors that occur due to deserialization or otherwise are returned.
	//
	// It is safe to call this method from multiple goroutines if the underlying [Handler] is.
	Execute(ctx context.Context, payload []byte) (*Result, error)
}

type typedHandlerFactory[T, TContext Data] struct {
	h Handler[T, TContext]
}

func (t *typedHandlerFactory[T, TContext]) Execute(ctx context.Context, payload []byte) (*Result, error) {
	var request Request[T, TContext]
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON failed: %w", err)
	}

	return t.h.Handle(ctx, request)
}

type handlerFunc[T, TContext Data] func(context.Context, Request[T, TContext]) (*Result, error)

func (h handlerFunc[T, TContext]) Handle(ctx context.Context, evt Request[T, TContext]) (*Result, error) {
	return h(ctx, evt)
}

func createHandlerFunc[T, TContext Data](f func(context.Context, Request[T, TContext]) (*Result, error)) Handler[T, TContext] {
	return handlerFunc[T, TContext](f)
}

// Handlers returns all registered handlers.
// It should not be modified directly, instead add new handlers only via [AddHandlerFunc].
func Handlers() map[string]HandlerFactory { return globalMux.handlers }
