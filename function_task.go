package utask

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type functionTask struct {
	opts options[functionTaskOptions]

	stdout io.Writer
	stderr io.Writer
	ret    chan error
}

func NewFunctionTask(opts ...FunctionTaskOption) (Task, error) {
	mergedOpts := options[functionTaskOptions]{
		specific: functionTaskOptions{},
	}
	for _, opt := range opts {
		if err := opt.apply(&mergedOpts); err != nil {
			return nil, err
		}
	}

	if mergedOpts.specific.fn == nil {
		return nil, errors.New("utask: no function given")
	}

	if mergedOpts.ctx == nil {
		mergedOpts.ctx = context.Background()
	}

	stdout := io.Discard
	if mergedOpts.stdout != nil {
		stdout = newNewLineWriter(mergedOpts.stdout)
	}
	stderr := io.Discard
	if mergedOpts.stderr != nil {
		stderr = newNewLineWriter(mergedOpts.stderr)
	}

	return &functionTask{
		opts:   mergedOpts,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

func (t *functionTask) Run() error {
	err := t.Start()
	if err != nil {
		return err
	}

	return t.Wait()
}

func (t *functionTask) String() string {
	return fmt.Sprintf("FunctionTask{fn:%p}", t.opts.specific.fn)
}

func (t *functionTask) Start() error {
	t.ret = make(chan error)

	go func() {
		err := t.opts.specific.fn(t.opts.ctx, t.stdout, t.stderr)
		if err != nil {
			_, _ = t.stderr.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		}
		t.ret <- err
	}()

	return nil
}

func (t *functionTask) Wait() error {
	if t.ret == nil {
		return errors.New("utask: not started")
	}
	return <-t.ret
}
