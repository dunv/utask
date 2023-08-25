package utask

import (
	"context"
	"io"
)

type FunctionTaskOptions struct {
	fn func(context.Context, io.Writer, io.Writer) error
}

var WithFunctionContext = WithContext[FunctionTaskOptions]
var WithFunctionMeta = WithMeta[FunctionTaskOptions]
var WithFunctionPrintStartAndEndInOutput = WithPrintStartAndEndInOutput[FunctionTaskOptions]
var WithFunctionStdout = WithStdout[FunctionTaskOptions]
var WithFunctionStderr = WithStderr[FunctionTaskOptions]
var WithFunctionCombinedOutput = WithCombinedOutput[FunctionTaskOptions]

func WithFunction(fn func(context.Context, io.Writer, io.Writer) error) TaskOption[FunctionTaskOptions] {
	return newFuncTaskOption(func(o *options[FunctionTaskOptions]) error {
		o.specific.fn = fn
		return nil
	})
}
