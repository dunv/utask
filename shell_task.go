package utask

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type shellTask struct {
	opts   options[shellTaskOptions]
	cmd    *exec.Cmd
	done   chan error
	waited bool
}

// Create a new shell task
func NewShellTask(opts ...ShellTaskOption) (Task, error) {
	mergedOpts := options[shellTaskOptions]{
		specific: shellTaskOptions{
			shellTermSignal: syscall.SIGTERM,
			// Bound output draining and allow graceful group termination.
			shellWaitDelay: 1 * time.Second,
		},
	}
	for _, opt := range opts {
		if err := opt.apply(&mergedOpts); err != nil {
			return nil, err
		}
	}

	if mergedOpts.specific.shellCommand == "" {
		return nil, errors.New("utask: no command given")
	}

	if mergedOpts.ctx == nil {
		mergedOpts.ctx = context.Background()
	}

	return &shellTask{opts: mergedOpts}, nil
}

// Runs the task and waits for it to complete
func (t *shellTask) Run() error {
	err := t.Start()
	if err != nil {
		return err
	}

	return t.Wait()
}

// Start the task, but don't wait for it to complete.
// Can be run in a go-routine if asynchroneous execution is desired.
func (t *shellTask) Start() error {
	if t.cmd != nil {
		return errors.New("utask: already started")
	}
	if err := t.opts.ctx.Err(); err != nil {
		return err
	}
	// The task owns cancellation and reaping. exec.CommandContext's watcher
	// can finish before output draining, and WaitDelay only kills the leader.
	cmd := exec.Command(t.opts.specific.shellCommand, t.opts.specific.shellArgs...)

	// use process-group-id as handle instead of process-id
	// set termSignal
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: t.opts.specific.shellTermSignal,
	}

	cmd.Dir = t.opts.specific.shellWorkingDir

	if t.opts.specific.shellEnv != nil {
		cmd.Env = t.opts.specific.shellEnv
	}

	output, err := newShellOutput(cmd, t.opts.stdout, t.opts.stderr)
	if err != nil {
		return err
	}

	// initial output (helps for debugging what is actually being run here)
	if t.opts.printStartAndEndInOutput {
		t.printStdOut("Running command '%s %s'", t.opts.specific.shellCommand, strings.Join(t.opts.specific.shellArgs, " "))
	}

	if err := cmd.Start(); err != nil {
		output.close()
		t.printStdErr("%s", err)
		return err
	}

	t.cmd = cmd
	t.done = make(chan error, 1)
	output.start()
	go func() { t.done <- t.waitShell(output) }()
	return nil
}

// Wait for the task to be completed. Can only be called once.
func (t *shellTask) Wait() error {
	if t.cmd == nil {
		return errors.New("utask: not started")
	}

	if t.waited {
		return errors.New("utask: Wait was already called")
	}
	t.waited = true
	if t.opts.printStartAndEndInOutput {
		defer t.printStdOut("Done executing")
	}

	err := <-t.done
	if err != nil {
		t.printStdErr("%s", err)
		return err
	}
	return nil
}

func (t *shellTask) String() string {
	return fmt.Sprintf("ShellTask{command:%s, args:%s}", t.opts.specific.shellCommand, strings.Join(t.opts.specific.shellArgs, " "))
}

func (t *shellTask) printStdOut(format string, args ...interface{}) {
	if t.opts.stdout != nil {
		fmt.Fprintf(t.opts.stdout, fmt.Sprintf("%s\n", format), args...)
	}
}

func (t *shellTask) printStdErr(format string, args ...interface{}) {
	if t.opts.stderr != nil {
		fmt.Fprintf(t.opts.stderr, fmt.Sprintf("%s\n", format), args...)
	}
}
