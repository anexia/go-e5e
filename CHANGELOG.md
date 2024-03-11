# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 2.1.0 - 2024-03-11

### Added
- Binary file API


## 2.0.1 - 2024-01-11

### Fixed
- Documentation fixes


## 2.0.0 - 2024-01-10

### Changed
- Set minimum Go version to 1.18.
- Rewritten the library to follow the Mux pattern of the `net/http` library.
  This ensures that there are no longer runtime errors for wrong entrypoint signatures, since everything
  needs to implement the new `e5e.Handler` interface now.
- Add support for cancellation using `context.Context`.
- Add support for strongly-typed request parameters, using the generics of Go 1.18.

A possible migration looks like this (old version first, new version second).

```go
package main

import (
  "go.anx.io/e5e"
)

type SumEventData struct {
  A int `json:"a,omitempty"`
  B int `json:"b,omitempty"`
}
type entrypoints struct{}

func (e entrypoints) Sum(d SumEventData, _ e5e.Context) (e5e.Result, error) {
  return e5e.Result{Data: d.A + d.B}, nil
}

func main() {
  e5e.Start(&entrypoints{})
}

```

```go
package main

import (
  "context"
  "log"
  "go.anx.io/e5e/v2"
)

type SumEventData struct {
  A int `json:"a,omitempty"`
  B int `json:"b,omitempty"`
}

func Sum(ctx context.Context, request e5e.Request[SumEventData, any]) (*e5e.Result, error) {
  d := request.Data()
  return &e5e.Result{Data: d.A + d.B}, nil
}

func main() {
  e5e.AddHandlerFunc("Sum", Sum)
  e5e.Start(context.Background())
}
```

### Added
- Table-driven tests for tests
- Add detailed documentation for all public types


## 1.2.1 - 2023-08-07

### Fixed
- Allow value `nil` for `Data` in `e5e.Result`


## 1.2.0 - 2022-08-30

### Changed
- Reimplemented with new binary runtime support (breaks support with previous custom runtime)
- Moved repository to https://github.com/anexia/go-e5e
- Use the Anexia vanity import path in the documentation


## 1.1.0 - 2020-08-24

### Changed
- Pass context and event variables with files instead of command line args


## 1.0.0 - 2020-08-07

### Added
- Initial release
