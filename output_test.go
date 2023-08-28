package utask

import (
	"bufio"
	"bytes"
	"io"
)

// helper for collecting and checking output
// in tests
type Output interface {
	io.Writer
	Lines() []string
}

type output struct {
	buf *bytes.Buffer
}

// create helper for collecting and checking output
// in tests
func NewOutput() *output {
	return &output{
		buf: bytes.NewBuffer([]byte{}),
	}
}

func (o *output) Write(p []byte) (n int, err error) {
	return o.buf.Write(p)
}

func (o *output) Lines() []string {
	output := []string{}
	sc := bufio.NewScanner(o.buf)
	for sc.Scan() {
		// when parsing fn-output, we always have a newline at the end
		if sc.Text() != "" {
			output = append(output, sc.Text())
		}
	}
	return output
}
