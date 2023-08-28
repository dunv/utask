package utask

import (
	"context"
	"io"
)

type functionTaskOptions struct {
	fn func(context.Context, io.Writer, io.Writer) error
}

// Options for a function task
type FunctionTaskOption taskOption[functionTaskOptions]

// Context to be supplied to the function
// Usually used for timeouts
var WithFunctionContext = withContext[functionTaskOptions]

// Print start and end of the function to stdout
var WithFunctionPrintStartAndEndInOutput = withPrintStartAndEndInOutput[functionTaskOptions]

// Write stdout on the given writer (default: os.Discard)
var WithFunctionStdout = withStdout[functionTaskOptions]

// Write stderr on the given writer (default: os.Discard)
var WithFunctionStderr = withStderr[functionTaskOptions]

// Write stdout and stderr on the given writer (default: os.Discard)
var WithFunctionCombinedOutput = withCombinedOutput[functionTaskOptions]

// Function to be executed.
//   - supplied context should be checked regularly
//   - a running function cannot be cancelled from the "outside", thus it is imperative
//     that the function check its context in order to be "timeoutable" or cancellable
//   - analogous to a shell command, stdout and stderr writers are supplied
func WithFunction(fn func(context.Context, io.Writer, io.Writer) error) FunctionTaskOption {
	return newFuncTaskOption(func(o *options[functionTaskOptions]) error {
		o.specific.fn = fn
		return nil
	})
}
