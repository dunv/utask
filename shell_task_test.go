package utask_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/dunv/utask"
	"github.com/stretchr/testify/require"
)

func TestShellTaskSimple(t *testing.T) {
	o := utask.NewOutput()
	task, err := utask.NewShellTask(
		utask.WithShellPrintStartAndEndInOutput(),
		utask.WithCommand("/bin/sh", "-c", "echo hallo"),
		utask.WithShellCombinedOutput(o),
	)
	require.NoError(t, err)
	require.NoError(t, task.Run())
	requireOutput(t, o,
		"Running command '/bin/sh -c echo hallo'",
		"hallo",
		"Done executing",
	)
}

func TestShellTaskInitErrorCombined(t *testing.T) {
	o := utask.NewOutput()
	task, err := utask.NewShellTask(
		utask.WithCommand("/bin/shNotExisting", "-c", "echo hallo"),
		utask.WithShellCombinedOutput(o),
	)
	require.NoError(t, err)
	require.Error(t, task.Run())
	requireOutput(t, o, "fork/exec /bin/shNotExisting: no such file or directory")
}

func TestShellTaskInitError(t *testing.T) {
	err, stdout, stderr := runShell("/bin/shNotExisting", []string{"-c", "echo hello"}, 2*time.Second, "", syscall.SIGTERM, 0)
	require.ErrorContains(t, err, "fork/exec /bin/shNotExisting: no such file or directory")
	requireOutput(t, stdout)
	requireOutput(t, stderr, "fork/exec /bin/shNotExisting: no such file or directory")
}

func TestShellTaskSuccessInTime(t *testing.T) {
	err, stdout, stderr := runShell("/bin/sh", []string{"-c", "echo hello"}, 2*time.Second, "", syscall.SIGTERM, 0)
	require.NoError(t, err)
	requireOutput(t, stdout, "hello")
	requireOutput(t, stderr)
}

func TestShellTaskSuccessNotInTimeTERM(t *testing.T) {
	err, stdout, stderr := runShell("/bin/sh", []string{"-c", "echo 1 && sleep .01 && echo 2 && sleep 10 && echo 3"}, 100*time.Millisecond, "", syscall.SIGTERM, 0)
	require.ErrorContains(t, err, "signal: terminated")
	requireOutput(t, stdout, "1", "2")
	requireOutput(t, stderr, "signal: terminated")
}

func TestShellTaskSuccessNotInTimeKILL(t *testing.T) {
	err, stdout, stderr := runShell("/bin/sh", []string{"-c", "echo 1 && sleep .01 && echo 2 && sleep 10 && echo 3"}, 100*time.Millisecond, "", syscall.SIGKILL, 0)
	require.ErrorContains(t, err, "signal: killed")
	requireOutput(t, stdout, "1", "2")
	requireOutput(t, stderr, "signal: killed")
}

func TestShellTaskError(t *testing.T) {
	err, stdout, stderr := runShell("/bin/sh", []string{"-c", "exit 1"}, 2*time.Second, "", syscall.SIGTERM, 0)
	require.ErrorContains(t, err, "exit status 1")
	requireOutput(t, stdout)
	requireOutput(t, stderr, "exit status 1")
}

func TestShellTaskEndlessWithIgnoreExitSignal(t *testing.T) {
	workingDir, err := filepath.Abs("ignoreAllSignals")
	require.NoError(t, err)
	executablePath := path.Join(workingDir, "ignoreAllSignals")
	err, stdout, stderr := runShell("go", []string{"build", "-o", executablePath}, 10*time.Second, workingDir, syscall.SIGTERM, 0)
	require.NoError(t, err)

	requireOutput(t, stdout)
	requireOutput(t, stderr)

	err, stdout, stderr = runShell("sh", []string{"-c", executablePath}, 100*time.Millisecond, "", syscall.SIGKILL, 0)
	require.ErrorContains(t, err, "signal: killed")
	time.Sleep(100 * time.Millisecond)

	// trapped signal might be in output
	out := stdout.Lines()
	require.Greater(t, len(out), 0)
	require.Less(t, len(out), 3)
	if len(out) == 1 {
		require.Equal(t, "still running", out[0])
	} else if len(out) == 2 {
		require.Equal(t, "still running", out[0])
		require.Equal(t, "trapped urgent I/O condition", out[1])
	}
	requireOutput(t, stderr, "signal: killed")
}

func TestShellTaskEndlessWithDetachedChildren(t *testing.T) {
	// waitDelay is set here to make the test faster (should default to 1s otherwise)
	err, stdout, stderr := runShell("sh", []string{"-c", "echo 1 && sleep 10000 &"}, 1000*time.Millisecond, "", syscall.SIGKILL, 100*time.Millisecond)
	require.ErrorContains(t, err, "exec: WaitDelay expired before I/O complete")
	requireOutput(t, stdout, "1")
	requireOutput(t, stderr, "exec: WaitDelay expired before I/O complete")
}

func TestShellTaskCancel(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	stdout := utask.NewOutput()
	stderr := utask.NewOutput()
	task, err := utask.NewShellTask(
		utask.WithShellContext(ctx),
		utask.WithCommand("/bin/sh", "-c", "echo 1 && sleep 1 && echo 2 && sleep 1 && echo 3"),
		utask.WithShellStdout(stdout),
		utask.WithShellStderr(stderr),
	)
	require.NoError(t, err)
	require.NoError(t, task.Start())
	time.Sleep(50 * time.Millisecond)
	cancel(fmt.Errorf("testCancel"))
	require.ErrorContains(t, task.Wait(), "signal: terminated")
	requireOutput(t, stdout, "1")
	requireOutput(t, stderr, "signal: terminated")
}

// Added for manual tests
func TestShellTaskAnsibleTimeout(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping test in CI environment")
	}

	require.NoError(t, os.WriteFile("ansible.yaml", []byte(`
---
- name: Change the working directory to somedir/ before executing the command
  hosts: localhost
  connection: local
  tasks:
    - ansible.builtin.shell: echo "test" && sleep 100
`), 0644))
	defer func() {
		os.RemoveAll("ansible.yaml")
	}()

	err, stdout, stderr := runShell("ansible-playbook", []string{"ansible.yaml"}, 1000*time.Millisecond, "", syscall.SIGKILL, 0)
	require.ErrorContains(t, err, "signal: killed")
	fmt.Println("stdout:", stdout.Lines())
	fmt.Println("stderr:", stderr.Lines())
}

// test-helper for running shell in one line
func runShell(cmd string, args []string, timeout time.Duration, workingDir string, termSignal syscall.Signal, waitDelay time.Duration) (error, utask.Output, utask.Output) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout := utask.NewOutput()
	stderr := utask.NewOutput()

	opts := []utask.TaskOption[utask.ShellTaskOptions]{
		utask.WithShellContext(ctx),
		utask.WithCommand(cmd, args...),
		utask.WithShellStdout(stdout),
		utask.WithShellStderr(stderr),
		utask.WithTermSignal(termSignal),
	}

	if workingDir != "" {
		opts = append(opts, utask.WithWorkingDir(workingDir))
	}

	if waitDelay > 0 {
		opts = append(opts, utask.WithWaitDelay(waitDelay))
	}

	task, err := utask.NewShellTask(opts...)
	if err != nil {
		return err, stdout, stderr
	}

	if err := task.Run(); err != nil {
		return err, stdout, stderr
	}
	return err, stdout, stderr
}

// test-helper for verifying output
func requireOutput(t *testing.T, o utask.Output, expected ...string) {
	output := o.Lines()
	require.Len(t, output, len(expected))
	for i, expectedLine := range expected {
		require.Equal(t, expectedLine, output[i])
	}
}
