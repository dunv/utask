package utask

import (
	"syscall"
	"time"
)

type ShellTaskOptions struct {
	shellCommand    string
	shellArgs       []string
	shellEnv        []string
	shellWorkingDir string
	shellTermSignal syscall.Signal
	shellWaitDelay  time.Duration
}

var WithShellContext = WithContext[ShellTaskOptions]
var WithShellMeta = WithMeta[ShellTaskOptions]
var WithShellPrintStartAndEndInOutput = WithPrintStartAndEndInOutput[ShellTaskOptions]
var WithShellStdout = WithStdout[ShellTaskOptions]
var WithShellStderr = WithStderr[ShellTaskOptions]
var WithShellCombinedOutput = WithCombinedOutput[ShellTaskOptions]

func WithCommand(command string, args ...string) TaskOption[ShellTaskOptions] {
	return newFuncTaskOption(func(o *options[ShellTaskOptions]) error {
		o.specific.shellCommand = command
		o.specific.shellArgs = args
		return nil
	})
}

func WithEnvironment(env []string) TaskOption[ShellTaskOptions] {
	return newFuncTaskOption(func(o *options[ShellTaskOptions]) error {
		o.specific.shellEnv = env
		return nil
	})
}

func WithWorkingDir(workingDir string) TaskOption[ShellTaskOptions] {
	return newFuncTaskOption(func(o *options[ShellTaskOptions]) error {
		o.specific.shellWorkingDir = workingDir
		return nil
	})
}

func WithTermSignal(termSignal syscall.Signal) TaskOption[ShellTaskOptions] {
	return newFuncTaskOption(func(o *options[ShellTaskOptions]) error {
		o.specific.shellTermSignal = termSignal
		return nil
	})
}

func WithWaitDelay(waitDelay time.Duration) TaskOption[ShellTaskOptions] {
	return newFuncTaskOption(func(o *options[ShellTaskOptions]) error {
		o.specific.shellWaitDelay = waitDelay
		return nil
	})
}
