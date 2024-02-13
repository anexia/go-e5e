package e5e_test

import (
	"context"

	"go.anx.io/e5e/v2"
)

func BinaryInverse(_ context.Context, request e5e.Request[e5e.File, any]) (*e5e.Result, error) {
	var outputBinary []byte

	inputBinary := request.Data().Bytes()
	for _, inputByte := range inputBinary {
		outputBinary = append(outputBinary, inputByte^255)
	}

	outputFile := &e5e.File{
		Name:        "output.blob",
		ContentType: "x-my-first-function/blob",
	}
	outputFile.Write(outputBinary)

	return &e5e.Result{
		Type: e5e.ResultDataTypeBinary,
		Data: outputFile,
	}, nil
}

func Example_binaryContent() {
	e5e.AddHandlerFunc("MyFunction", BinaryInverse)
	e5e.Start(context.Background())
}
