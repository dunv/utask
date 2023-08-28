package utask

import (
	"context"
	"io"
)

type options[T specificOptions] struct {
	ctx                      context.Context
	printStartAndEndInOutput bool
	stdout                   io.Writer
	stderr                   io.Writer
	specific                 T
}

type specificOptions interface {
	functionTaskOptions | shellTaskOptions
}

type taskOption[T specificOptions] interface {
	apply(*options[T]) error
}
type funcTaskOption[T specificOptions] struct {
	f func(*options[T]) error
}

// nolint:unused
func (fdo *funcTaskOption[T]) apply(do *options[T]) error {
	return fdo.f(do)
}

func newFuncTaskOption[T specificOptions](f func(*options[T]) error) *funcTaskOption[T] {
	return &funcTaskOption[T]{f: f}
}

func withContext[T specificOptions](ctx context.Context) taskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.ctx = ctx
		return nil
	})
}

func withPrintStartAndEndInOutput[T specificOptions]() taskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.printStartAndEndInOutput = true
		return nil
	})
}

func withStdout[T specificOptions](w io.Writer) taskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stdout = w
		return nil
	})
}

func withStderr[T specificOptions](w io.Writer) taskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stderr = w
		return nil
	})
}

func withCombinedOutput[T specificOptions](w io.Writer) taskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stdout = w
		o.stderr = w
		return nil
	})
}
