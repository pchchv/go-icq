package wire

import "io"

// errWriter is a writer that always returns an error.
type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) {
	return 0, io.EOF
}
