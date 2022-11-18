go-e5e
======

[![PkgGoDev](https://pkg.go.dev/badge/go.anx.io/e5e)](https://pkg.go.dev/go.anx.io/e5e)
[![Build Status](https://github.com/anexia/go-e5e/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/anexia/go-e5e/actions/?query=workflow%3Atest)
[![codecov](https://codecov.io/gh/anexia/go-e5e/branch/main/graph/badge.svg)](https://codecov.io/gh/anexia/go-e5e)
[![Go Report Card](https://goreportcard.com/badge/go.anx.io/e5e)](https://goreportcard.com/report/go.anx.io/e5e)

go-e5e is a support library to help Go developers build Anexia e5e functions.

# Install

With a [correctly configured](https://go.dev/doc/install) Go toolchain:

```sh
go get -u go.anx.io/e5e
```

# Getting started

```go
package main

import (
	"runtime"
	
	"go.anx.io/e5e"
)

type SumEventData struct {
	A int `json:"a"`
	B int `json:"b"`
}

type SumEvent struct {
	e5e.Event
	Data SumEventData `json:"data,omitempty"`
}

// Using a custom `e5e.Context` class is optional and only needed to access the `Data` attribute on the
// context. This attribute may be used to get the return value of an authorizer function, for example. If access
// to the `Data` attribute is not needed, the `e5e.Context` type can be used on the entrypoint directly.

type SumContextData struct {
	AuthKey string `json:"auth_key"`
}

type SumContext struct {
	e5e.Context
	Data SumContextData `json:"data,omitempty"`
}

type entrypoints struct{}

func (f *entrypoints) MyEntrypoint(event SumEvent, context SumContext) (e5e.Result, error) {
	return e5e.Result{
		Status: 200,
		ResponseHeaders: map[string]string{
			"x-custom-response-header": "This is a custom response header",
		},
		Data: map[string]interface{}{
			"sum": event.Data.A + event.Data.B,
			"version": runtime.Version(),
		},
	}, nil
}

func main() {
	e5e.Start(&entrypoints{})
}
```

# List of developers

* Andreas Stocker <AStocker@anexia-it.com>, Lead Developer
* Patrick Taibel <PTaibel@anexia-it.com>, Developer
