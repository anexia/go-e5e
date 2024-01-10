package e5e_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type stdioFake struct {
	// faking stdin
	inOrig *os.File

	// faking stdout
	outOrig   *os.File
	outReader *os.File
	outCh     chan []byte

	// faking stderr
	errOrig   *os.File
	errReader *os.File
	errCh     chan []byte

	restored atomic.Bool
}

// ReadAndRestore collects all captured stdout and returns it; it also restores
// os.Stdin and os.Stdout to their original values.
func (sf *stdioFake) ReadAndRestore() (stdout string, stderr string) {
	if sf.restored.Load() {
		return
	}
	defer sf.restored.Store(true)

	os.Stdin.Close()
	os.Stdin = sf.inOrig

	// Close the writer side of the faked stdout pipe. This signals to the
	// background goroutine that it should exit.
	os.Stdout.Close()
	out := <-sf.outCh
	os.Stdout = sf.outOrig

	// Close the writer side of the faked stdout pipe. This signals to the
	// background goroutine that it should exit.
	os.Stderr.Close()
	errOut := <-sf.errCh
	os.Stderr = sf.errOrig

	if sf.outReader != nil {
		sf.outReader.Close()
	}

	return string(out), string(errOut)
}

func redirectStdio(t *testing.T, stdin string) *stdioFake {
	t.Helper()

	stdinFile := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(stdinFile, []byte(stdin), 0o644); err != nil {
		t.Fatalf("creating temporary stdin failed: %v", err)
	}

	origStdin := os.Stdin
	os.Stdin, _ = os.Open(stdinFile)

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe for fake stdout failed: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = stdoutWriter
	outCh := make(chan []byte)

	// This goroutine reads stdout into a buffer in the background.
	go func() {
		var b bytes.Buffer
		if _, err := io.Copy(&b, stdoutReader); err != nil {
			log.Fatalf("reading from stdout failed: %v", err)
		}
		outCh <- b.Bytes()
	}()

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe for fake stderr failed: %v", err)
	}

	origStderr := os.Stderr
	os.Stderr = stderrWriter
	errCh := make(chan []byte)

	// This goroutine reads stdout into a buffer in the background.
	go func() {
		var b bytes.Buffer
		if _, err := io.Copy(&b, stderrReader); err != nil {
			log.Fatalf("reading from stderr failed: %v", err)
		}
		errCh <- b.Bytes()
	}()

	fake := &stdioFake{
		inOrig: origStdin,

		outOrig:   origStdout,
		outReader: stdoutReader,
		outCh:     outCh,

		errOrig:   origStderr,
		errReader: stderrReader,
		errCh:     errCh,
	}
	t.Cleanup(func() {
		fake.ReadAndRestore()
	})

	return fake
}
