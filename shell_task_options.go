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

type ShellTaskOption taskOption[shellTaskOptions]

var WithShellContext = withContext[shellTaskOptions]
var WithShellMeta = withMeta[shellTaskOptions]
var WithShellPrintStartAndEndInOutput = withPrintStartAndEndInOutput[shellTaskOptions]
var WithShellStdout = withStdout[shellTaskOptions]
var WithShellStderr = withStderr[shellTaskOptions]
var WithShellCombinedOutput = withCombinedOutput[shellTaskOptions]

func WithShellCommand(command string, args ...string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellCommand = command
		o.specific.shellArgs = args
		return nil
	})
}

func WithShellEnvironment(env []string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellEnv = env
		return nil
	})
}

func WithShellWorkingDir(workingDir string) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellWorkingDir = workingDir
		return nil
	})
}

func WithShellTermSignal(termSignal syscall.Signal) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellTermSignal = termSignal
		return nil
	})
}

func WithShellWaitDelay(waitDelay time.Duration) ShellTaskOption {
	return newFuncTaskOption(func(o *options[shellTaskOptions]) error {
		o.specific.shellWaitDelay = waitDelay
		return nil
	})
}
