package utask

import "io"

type newLineWriter struct {
	w io.Writer
}

func newNewLineWriter(w io.Writer) *newLineWriter {
	return &newLineWriter{w: w}
}

func (w *newLineWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(append(p, '\n'))
}
