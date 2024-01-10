go-e5e
======

[![PkgGoDev](https://pkg.go.dev/badge/go.anx.io/e5e/v2)](https://pkg.go.dev/go.anx.io/e5e/v2)
[![Build Status](https://github.com/anexia/go-e5e/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/anexia/go-e5e/actions/?query=workflow%3Atest)
[![codecov](https://codecov.io/gh/anexia/go-e5e/branch/main/graph/badge.svg)](https://codecov.io/gh/anexia/go-e5e)
[![Go Report Card](https://goreportcard.com/badge/go.anx.io/e5e/v2)](https://goreportcard.com/report/go.anx.io/e5e/v2)

go-e5e is a support library to help Go developers build Anexia e5e functions.

## Install

With a [correctly configured](https://go.dev/doc/install) Go toolchain:

```sh
go get -u go.anx.io/e5e/v2
```

## Getting started

```go
package main

import (
	"context"
	
	"go.anx.io/e5e/v2"
)

type SumData struct {
	A int `json:"a"`
	B int `json:"b"`
}

func Sum(ctx context.Context, r e5e.Request[SumData, any]) (*e5e.Result, error) {
	result := r.Data().A + r.Data().B
	return &e5e.Result{
		Status: 200,
		ResponseHeaders: map[string]string{
			"x-custom-response-header": "This is a custom response header",
		},
		Data: result,
	}, nil
}

func main() {
	e5e.AddHandlerFunc("Sum", Sum)
	e5e.Start(context.Background())
}

```

## List of developers

* Andreas Stocker <AStocker@anexia-it.com>, Lead Developer
* Patrick Taibel <PTaibel@anexia-it.com>, Developer
* Jasmin Oster <joster@anexia.com>, Developer

## Links

<!-- Those links are fetched by pkg.go.dev and displayed in the sidebar. -->

- [e5e Documentation](https://engine.anexia-it.com/docs/en/module/e5e/)
