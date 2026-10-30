package utask

import (
	"errors"
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

// WithShellTermSignal sets the initial group-cleanup signal (default: SIGTERM).
// Cleanup follows cancellation, unsuccessful exit or an output error/timeout.
// A positive WithShellWaitDelay escalates to group-wide SIGKILL after the grace
// interval, even when the command has exited. SIGKILL skips the grace interval.
func WithShellTermSignal(termSignal syscall.Signal) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellTermSignal = termSignal
		return nil
	})
}

// WithShellWaitDelay sets the output-drain timeout and group-termination grace
// interval (default: 1s). These are separate phases: a drain timeout can trigger
// cleanup, and cleanup is followed by a final bounded drain. Zero disables drain
// timeouts and SIGKILL escalation; uncooperative children can then outlive the
// task or keep it waiting indefinitely. Negative durations are rejected.
func WithShellWaitDelay(waitDelay time.Duration) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		if waitDelay < 0 {
			return errors.New("utask: shell wait delay must not be negative")
		}
		o.specific.shellWaitDelay = waitDelay
		return nil
	})
}
