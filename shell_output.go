package utask

import (
	"io"
	"os"
	"os/exec"
	"reflect"
	"sync"
)

// Own the pipes so draining can finish before the group leader is reaped.
// Passing files (including nil) through preserves os/exec's no-copy behavior.
type shellOutput struct {
	streams []shellStream
	done    chan error
}

type shellStream struct {
	reader *os.File
	writer *os.File
	target io.Writer
}

func newShellOutput(cmd *exec.Cmd, stdout, stderr io.Writer) (*shellOutput, error) {
	o := &shellOutput{done: make(chan error, 1)}
	var err error
	cmd.Stdout, err = o.descriptor(stdout)
	if err == nil {
		if stdout != nil && reflect.ValueOf(stdout).Comparable() && stdout == stderr {
			// One pipe preserves stdout/stderr ordering for combined output.
			cmd.Stderr = cmd.Stdout
		} else {
			cmd.Stderr, err = o.descriptor(stderr)
		}
	}
	if err != nil {
		o.close()
		return nil, err
	}
	return o, nil
}

func (o *shellOutput) descriptor(w io.Writer) (io.Writer, error) {
	if w == nil {
		return nil, nil
	}
	if f, ok := w.(*os.File); ok {
		return f, nil
	}
	r, p, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	o.streams = append(o.streams, shellStream{reader: r, writer: p, target: w})
	return p, nil
}

func (o *shellOutput) start() {
	results := make(chan error, len(o.streams))
	// Combined-output writers need serialized calls, just as with os/exec.
	var mu sync.Mutex
	for _, stream := range o.streams {
		_ = stream.writer.Close()
		go func(s shellStream) {
			_, err := io.Copy(&shellOutputWriter{mu: &mu, target: s.target}, s.reader)
			_ = s.reader.Close()
			results <- err
		}(stream)
	}
	go func() {
		var first error
		for range o.streams {
			if err := <-results; first == nil {
				first = err
			}
		}
		o.done <- first
	}()
}

func (o *shellOutput) close() {
	for _, s := range o.streams {
		_ = s.writer.Close()
		_ = s.reader.Close()
	}
}

type shellOutputWriter struct {
	mu     *sync.Mutex
	target io.Writer
}

func (w *shellOutputWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.target.Write(p)
}
