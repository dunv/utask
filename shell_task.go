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
	opts options[shellTaskOptions]
	cmd  *exec.Cmd
}

// Create a new shell task
func NewShellTask(opts ...ShellTaskOption) (Task, error) {
	mergedOpts := options[shellTaskOptions]{
		specific: shellTaskOptions{
			shellTermSignal: syscall.SIGTERM,
			// set a default wait-delay, so a task will always finish after the
			// context is canceled (by default 1s after the context is canceled)
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
	cmd := exec.CommandContext(t.opts.ctx, t.opts.specific.shellCommand, t.opts.specific.shellArgs...)

	// use process-group-id as handle instead of process-id
	// set termSignal
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: t.opts.specific.shellTermSignal,
	}

	// override cancelFunc so the whole processGroup gets terminated
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, t.opts.specific.shellTermSignal)
	}

	cmd.Dir = t.opts.specific.shellWorkingDir

	if t.opts.specific.shellEnv != nil {
		cmd.Env = t.opts.specific.shellEnv
	}

	if t.opts.stdout != nil {
		cmd.Stdout = t.opts.stdout
	}

	if t.opts.stderr != nil {
		cmd.Stderr = t.opts.stderr
	}

	if t.opts.specific.shellWaitDelay > 0 {
		cmd.WaitDelay = t.opts.specific.shellWaitDelay
	}

	// save cmd to task-object (in case Start() + Wait() is used)
	t.cmd = cmd

	// initial output (helps for debugging what is actually being run here)
	if t.opts.printStartAndEndInOutput {
		t.printStdOut("Running command '%s %s'", t.opts.specific.shellCommand, strings.Join(t.opts.specific.shellArgs, " "))
	}

	err := cmd.Start()
	if err != nil {
		t.printStdErr(err.Error())
		return err
	}

	return nil
}

// Wait for the task to be completed. Can only be called once.
func (t *shellTask) Wait() error {
	if t.cmd == nil {
		return errors.New("utask: not started")
	}

	if t.opts.printStartAndEndInOutput {
		defer func() {
			t.printStdOut("Done executing")
		}()
	}

	err := t.cmd.Wait()
	if err != nil {
		t.printStdErr(err.Error())
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
