package utask

import (
	"context"
	"io"
)

type options[T specificOptions] struct {
	ctx                      context.Context
	meta                     interface{}
	printStartAndEndInOutput bool
	stdout                   io.Writer
	stderr                   io.Writer
	specific                 T
}

type specificOptions interface {
	FunctionTaskOptions | ShellTaskOptions
}

type TaskOption[T specificOptions] interface {
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

func WithContext[T specificOptions](ctx context.Context) TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.ctx = ctx
		return nil
	})
}

func WithMeta[T specificOptions](meta interface{}) TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.meta = meta
		return nil
	})
}

func WithPrintStartAndEndInOutput[T specificOptions]() TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.printStartAndEndInOutput = true
		return nil
	})
}

func WithStdout[T specificOptions](w io.Writer) TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stdout = w
		return nil
	})
}

func WithStderr[T specificOptions](w io.Writer) TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stderr = w
		return nil
	})
}

func WithCombinedOutput[T specificOptions](w io.Writer) TaskOption[T] {
	return newFuncTaskOption(func(o *options[T]) error {
		o.stdout = w
		o.stderr = w
		return nil
	})
}
