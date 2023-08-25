package utask_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/dunv/utask"
	"github.com/stretchr/testify/require"
)

func TestFunctionSuccess(t *testing.T) {
	err, stdout, stderr := runFunction(func(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
		if _, err := stdout.Write([]byte("TestOutput")); err != nil {
			return err
		}
		if _, err := stderr.Write([]byte("TestErr")); err != nil {
			return err
		}
		return nil
	}, 2*time.Second)

	require.NoError(t, err)
	requireOutput(t, stdout, "TestOutput")
	requireOutput(t, stderr, "TestErr")
}

func TestFunctionError(t *testing.T) {
	err, stdout, stderr := runFunction(func(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
		if _, err := stdout.Write([]byte("TestOutput")); err != nil {
			return err
		}
		if _, err := stderr.Write([]byte("TestErr")); err != nil {
			return err
		}
		return errors.New("fail on purpose")
	}, 2*time.Second)

	require.ErrorContains(t, err, "fail on purpose")
	requireOutput(t, stdout, "TestOutput")
	requireOutput(t, stderr, "TestErr", "fail on purpose")
}

func TestFunctionTimeoutSuccess(t *testing.T) {
	err, stdout, stderr := runFunction(func(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
		select {
		case <-time.After(2 * time.Second):
			_, _ = stdout.Write([]byte("TestOutputAfterTimeout (this should not be printed)"))
			return nil
		case <-ctx.Done():
			_, _ = stdout.Write([]byte("Context done"))
			return nil
		}
	}, 100*time.Millisecond)

	require.NoError(t, err)
	requireOutput(t, stdout, "Context done")
	requireOutput(t, stderr)
}

func TestFunctionTimeoutError(t *testing.T) {
	err, stdout, stderr := runFunction(func(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
		select {
		case <-time.After(2 * time.Second):
			_, _ = stdout.Write([]byte("TestOutputAfterTimeout (this should not be printed)"))
			return nil
		case <-ctx.Done():
			_, _ = stdout.Write([]byte("Context done"))
			return ctx.Err()
		}
	}, 100*time.Millisecond)

	require.ErrorContains(t, err, "context deadline exceeded")
	requireOutput(t, stdout, "Context done")
	requireOutput(t, stderr, "context deadline exceeded")
}

func TestFunctionTaskCancel(t *testing.T) {
	fn := func(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
		start := time.Now()
		for {
			// output
			_, _ = stdout.Write([]byte("running"))

			// if function gets 2 seconds to run: exit with success
			if time.Since(start) > 2*time.Second {
				return nil
			}

			// check for cancelled context "every 100ms"
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(20 * time.Millisecond):
				continue
			}
		}
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	stdout := utask.NewOutput()
	stderr := utask.NewOutput()
	task, err := utask.NewFunctionTask(
		utask.WithFunctionContext(ctx),
		utask.WithFunction(fn),
		utask.WithFunctionStdout(stdout),
		utask.WithFunctionStderr(stderr),
	)
	require.NoError(t, err)
	require.NoError(t, task.Start())
	time.Sleep(50 * time.Millisecond)
	cancel(errors.New("cancel on purpose"))
	require.ErrorContains(t, task.Wait(), "context canceled")

	requireOutput(t, stdout, "running", "running", "running")
	requireOutput(t, stderr, "context canceled")
}

// test-helper for running shell in one line
func runFunction(fn func(context.Context, io.Writer, io.Writer) error, timeout time.Duration) (error, utask.Output, utask.Output) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout := utask.NewOutput()
	stderr := utask.NewOutput()

	task, err := utask.NewFunctionTask(
		utask.WithFunctionContext(ctx),
		utask.WithFunction(fn),
		utask.WithFunctionStdout(stdout),
		utask.WithFunctionStderr(stderr),
	)
	if err != nil {
		return err, stdout, stderr
	}

	if err := task.Run(); err != nil {
		return err, stdout, stderr
	}
	return err, stdout, stderr
}
