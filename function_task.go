package utask

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type functionTask struct {
	opts options[FunctionTaskOptions]

	stdout io.Writer
	stderr io.Writer
	ret    chan error
}

func NewFunctionTask(opts ...TaskOption[FunctionTaskOptions]) (Task, error) {
	mergedOpts := options[FunctionTaskOptions]{
		specific: FunctionTaskOptions{},
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

// func (t *shellTask) runFunction() error {
// 	// functionOutputChannel := make(chan string)
// 	// var ctx context.Context

// 	// t.cancelLock.Lock()
// 	// ctx, t.cancelFunc = context.WithTimeout(context.Background(), t.opts.timeout)
// 	// t.cancelLock.Unlock()

// 	// go func(functionOutputChannel chan string) {
// 	// 	for output := range functionOutputChannel {
// 	// 		t.addOutput(TASK_OUTPUT_STDOUT, output)
// 	// 	}
// 	// }(functionOutputChannel)

// 	// go func(ctx context.Context, functionOutputChannel chan string) {
// 	// 	t.markAsInProgress()

// 	// 	// We are not adding a timeout here, so if a function runs indefinitely this is blocking as well.
// 	// 	// Canceling needs to be handled within function
// 	// 	exitCode := t.opts.fn(ctx, functionOutputChannel)

// 	// 	close(functionOutputChannel)

// 	// 	// If context has timed out -> set exitCode manually and add output
// 	// 	if ctx.Err() != nil {
// 	// 		exitCode = -1
// 	// 		t.addOutput(TASK_OUTPUT_STDERR, ctx.Err().Error())
// 	// 	}

// 	// 	if exitCode == 0 {
// 	// 		t.markAsSuccessful()
// 	// 	} else {
// 	// 		t.markAsFailed(exitCode, ctx.Err())
// 	// 	}
// 	// }(ctx, functionOutputChannel)

// 	return nil
// }
