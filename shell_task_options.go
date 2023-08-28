package utask

import (
	"syscall"
	"time"
)

type shellTaskOptions struct {
	shellCommand    string
	shellArgs       []string
	shellEnv        []string
	shellWorkingDir string
	shellTermSignal syscall.Signal
	shellWaitDelay  time.Duration
}

// Options for a shell task
type ShellTaskOption taskOption[shellTaskOptions]

// Context to be supplied to the function
// Usually used for timeouts
var WithShellContext = withContext[shellTaskOptions]

// Print start and end of the execution to stdout
var WithShellPrintStartAndEndInOutput = withPrintStartAndEndInOutput[shellTaskOptions]

// Write stdout on the given writer (default: os.Discard)
var WithShellStdout = withStdout[shellTaskOptions]

// Write stderr on the given writer (default: os.Discard)
var WithShellStderr = withStderr[shellTaskOptions]

// Write stdout and stderr on the given writer (default: os.Discard)
var WithShellCombinedOutput = withCombinedOutput[shellTaskOptions]

// Shell command to be executed.
func WithShellCommand(command string, args ...string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellCommand = command
		o.specific.shellArgs = args
		return nil
	})
}

// Environment variables to be set before executing the command
func WithShellEnvironment(env []string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellEnv = env
		return nil
	})
}

// Working directory to be set before executing the command
func WithShellWorkingDir(workingDir string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellWorkingDir = workingDir
		return nil
	})
}

// Term signal to be sent to the process-group-id of the command (default: SIGTERM)
func WithShellTermSignal(termSignal syscall.Signal) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellTermSignal = termSignal
		return nil
	})
}

// Time to wait after sending the term signal before sending the kill signal (default: 1s)
func WithShellWaitDelay(waitDelay time.Duration) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellWaitDelay = waitDelay
		return nil
	})
}
