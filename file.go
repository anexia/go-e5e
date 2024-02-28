package e5e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// File contains information about a received or sent file.
// It is commonly used with "mixed" or "binary" requests/responses.
type File struct {
	// The contents of the file, encoded in [Charset].
	content []byte

	// The type of this binary, usually just "binary".
	Type string `json:"type"`

	// The size of the file in bytes.
	// If it cannot be determined reliably, just leave it at the default value.
	SizeInBytes int64 `json:"size,omitempty"`

	// The optional filename of the file.
	Name string `json:"name,omitempty"`

	// The content type of the file.
	// For responses, the Content-Type heaader is set automatically be the E5E engine to this value.
	ContentType string `json:"content_type,omitempty"`

	// The charset of the file. Should be set to the recommended value "utf-8".
	Charset string `json:"charset,omitempty"`
}

// SetText sets the content of this file to the encoded version of text.
// It further enforces the content type to "text/plain". The file size and the charset are set,
// if they haven't been set already modified by the user.
func (f *File) SetPlainText(text string) error {
	_, err := f.Write([]byte(text))
	if err != nil {
		return err
	}
	f.ContentType = "text/plain"
	return nil
}

// Bytes returns the raw bytes of this file.
func (f File) Bytes() []byte { return f.content }

// Read implements io.Reader.
func (f File) Read(p []byte) (n int, err error) { return copy(p, f.content), io.EOF }

// Write implements io.Writer.
// It further sets the content type to the output of [http.DetectContentType],
// the file size and the charset, if none of those properties have been set before.
func (f *File) Write(p []byte) (n int, err error) {
	// Set a copy of the slice as the content, so we don't keep
	// a reference to the original.
	f.content = p[:]
	if f.Charset == "" {
		f.Charset = "utf-8"
	}
	if f.ContentType == "" {
		// If the content type appends the charset, we remove it.
		// This happens for content types like "text/plain; charset=utf-8"
		f.ContentType, _, _ = strings.Cut(http.DetectContentType(p), "; ")
	}
	if f.SizeInBytes == 0 {
		f.SizeInBytes = int64(len(p))
	}
	return len(p), nil
}

// rawFile describes the structure that we receive from e5e.
// It is just used for internal decoding.
type rawFile struct {
	Base64Encoded   string `json:"binary"`
	Type            string `json:"type"`
	FileSizeInBytes int64  `json:"size,omitempty"`
	Filename        string `json:"name,omitempty"`
	ContentType     string `json:"content_type,omitempty"`
	Charset         string `json:"charset,omitempty"`
}

// MarshalJSON implements json.Marshaler.
func (f File) MarshalJSON() ([]byte, error) {
	if f.Type == "" {
		f.Type = "binary"
	}

	file := rawFile{
		Base64Encoded:   base64.StdEncoding.EncodeToString(f.content),
		Type:            f.Type,
		FileSizeInBytes: f.SizeInBytes,
		Filename:        f.Name,
		ContentType:     f.ContentType,
		Charset:         f.Charset,
	}
	return json.Marshal(file)
}

// UnmarshalJSON implements json.Unmarshaler.
func (f *File) UnmarshalJSON(data []byte) error {
	var file rawFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	fileBytes, err := base64.StdEncoding.DecodeString(file.Base64Encoded)
	if err != nil {
		return fmt.Errorf("%q attribute does not contain a valid base64 string: %w", "binary", err)
	}

	f.content = fileBytes
	f.Type = file.Type
	f.SizeInBytes = file.FileSizeInBytes
	f.Name = file.Filename
	f.ContentType = file.ContentType
	f.Charset = file.Charset
	return nil
}

// compile-time check for certain interfaces
var _ io.Reader = File{}
var _ io.Writer = &File{}
var _ json.Unmarshaler = &File{}
var _ json.Marshaler = File{}
