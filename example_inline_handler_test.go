package e5e_test

import (
	"context"

	"go.anx.io/e5e/v2"
)

func Example_inlineHandler() {
	e5e.AddHandlerFunc("Reverse", func(ctx context.Context, r e5e.Request[string, any]) (*e5e.Result, error) {
		runes := []rune(r.Data())
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}

		return &e5e.Result{Data: string(runes)}, nil
	})
	e5e.Start(context.Background())
}
