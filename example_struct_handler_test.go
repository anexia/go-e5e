package e5e_test

import (
	"context"

	"go.anx.io/e5e/v2"
)

type SumData struct {
	A int `json:"a"`
	B int `json:"b"`
}

type SumHandler struct{}

func (s SumHandler) Handle(ctx context.Context, r e5e.Request[SumData, any]) (*e5e.Result, error) {
	result := r.Data().A + r.Data().B
	return &e5e.Result{Data: result}, nil
}

func Example_structHandler() {
	e5e.AddHandlerFunc("Sum", SumHandler{}.Handle)
	e5e.Start(context.Background())
}
