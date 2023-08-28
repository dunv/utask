package utask

import (
	"context"
	"io"
)

type functionTaskOptions struct {
	fn func(context.Context, io.Writer, io.Writer) error
}

type FunctionTaskOption taskOption[functionTaskOptions]

var WithFunctionContext = withContext[functionTaskOptions]
var WithFunctionMeta = withMeta[functionTaskOptions]
var WithFunctionPrintStartAndEndInOutput = withPrintStartAndEndInOutput[functionTaskOptions]
var WithFunctionStdout = withStdout[functionTaskOptions]
var WithFunctionStderr = withStderr[functionTaskOptions]
var WithFunctionCombinedOutput = withCombinedOutput[functionTaskOptions]

func WithFunction(fn func(context.Context, io.Writer, io.Writer) error) FunctionTaskOption {
	return newFuncTaskOption(func(o *options[functionTaskOptions]) error {
		o.specific.fn = fn
		return nil
	})
}
