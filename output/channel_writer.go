package output

import (
	"bufio"
	"bytes"
	"strings"
)

// Buffered writer for output
//   - collects all bytes
//   - if a line is terminated with \n, publish the line (without \n) to the outputChannel
type channelWriter struct {
	outChan chan<- string
	w       *bytes.Buffer
	r       *bufio.Reader
}

func (s channelWriter) Write(p []byte) (n int, err error) {
	if n, err := s.w.Write(p); err != nil {
		return n, err
	}

	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			// put it back in the buffer if there is no \n
			s.w.Write([]byte(line))
			return len(p), nil
		}

		if len(line) > 1 { // '\n' is included in line
			s.outChan <- strings.TrimSpace(string(line))
		}
	}
}

// Creates a buffered writer for collecting output
//   - collects all bytes
//   - if a line is terminated with \n, publish the line (without \n) to outChan
func NewChannelWriter(outChan chan<- string) channelWriter {
	w := bytes.NewBuffer([]byte{})
	return channelWriter{
		outChan: outChan,
		w:       w,
		r:       bufio.NewReader(w),
	}
}
