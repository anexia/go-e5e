package e5e_test

import (
	_ "embed"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"go.anx.io/e5e/v2"
)

//go:embed testdata/binary_request_with_multiple_files.json
var binaryRequestWithMultipleFiles []byte

func TestFile(t *testing.T) {
	t.Parallel()
	t.Run("SetText encodes the content properly", func(t *testing.T) {
		t.Parallel()
		f := &e5e.File{}
		_ = f.SetPlainText("Hello world!")

		Equal(t, "utf-8", f.Charset, "Charset does not match")
		Equal(t, 12, int(f.SizeInBytes), "file size does not match")
		Equal(t, "text/plain", f.ContentType, "content type does not match")

		var encodedBytes = []byte{72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 33}
		DeepEqual(t, encodedBytes, f.Bytes(), "bytes do not match")
	})
	t.Run("JSON serialization matches expectation", func(t *testing.T) {
		t.Parallel()
		var expect = `{"binary":"SGVsbG8gd29ybGQh","type":"binary","size":12,"content_type":"text/plain","charset":"utf-8"}`

		f := &e5e.File{}
		_, err := f.Write([]byte("Hello world!"))
		if err != nil {
			t.Errorf("expected no write error, got: %v", err)
		}

		// One test with the pointer
		actual, err := json.Marshal(f)
		if err != nil {
			t.Errorf("JSON marshalling failed: %v", err)
		}

		Equal(t, expect, string(actual), "JSON does not match")

		// And one without it
		actual, err = json.Marshal(*f)
		if err != nil {
			t.Errorf("JSON marshalling failed: %v", err)
		}

		Equal(t, expect, string(actual), "JSON does not match")
	})
	t.Run("JSON deserialization works", func(t *testing.T) {
		t.Parallel()
		var expected = e5e.File{
			Type:        "binary",
			SizeInBytes: 12,
			Name:        "my-file-1.name",
			ContentType: "application/my-content-type-1",
			Charset:     "utf-8",
		}
		expected.SetPlainText("Hello world!")
		expected.ContentType = "application/my-content-type-1"

		const input = `{
				"binary": "SGVsbG8gd29ybGQh",
				"type": "binary",
				"name": "my-file-1.name",
				"size": 12,
				"content_type": "application/my-content-type-1",
				"charset": "utf-8"
			}`
		var actual e5e.File
		if err := json.Unmarshal([]byte(input), &actual); err != nil {
			t.Errorf("JSON unmarshaling failed: %v", err)
		}
		DeepEqual(t, expected, actual, "files do not match")
	})
	t.Run("original slice is ignored", func(t *testing.T) {
		var original = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
		var modified = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
		f := &e5e.File{}
		n, _ := f.Write(modified)
		Equal(t, 9, n, "written bytes do not match")
		modified = append(modified, 10)
		DeepEqual(t, f.Bytes(), original, "slice got passed by reference")
	})
	t.Run("request can be deserialized", func(t *testing.T) {
		request := e5e.Request[[]e5e.File, any]{}
		if err := json.Unmarshal(binaryRequestWithMultipleFiles, &request); err != nil {
			t.Errorf("JSON unmarshaling failed: %v", err)
		}

		Equal(t, 2, len(request.Data()), "expected two files")
		for _, file := range request.Data() {
			Equal(t, "binary", file.Type, "file type does not match")
			Equal(t, 12, file.SizeInBytes, "file size does not match")
			Equal(t, "utf-8", file.Charset, "charset does not match")
			if !strings.HasPrefix(file.ContentType, "application/my-content-type") {
				t.Errorf("invalid content type prefix, got: %s", file.ContentType)
			}
			if !strings.HasPrefix(file.Name, "my-file-") {
				t.Errorf("invalid name prefix, got: %s", file.Name)
			}
		}
	})
	t.Run("file can be read", func(t *testing.T) {
		t.Parallel()
		file := &e5e.File{}
		if err := file.SetPlainText("Hello world!"); err != nil {
			t.Errorf("setting file content failed: %v", err)
		}

		var buf strings.Builder
		n, err := io.Copy(&buf, file)
		if err != nil {
			t.Errorf("copying failed: %v", err)
		}
		Equal(t, 12, n, "read bytes do not match")
		Equal(t, "Hello world!", buf.String(), "file content does not match")
	})
}
