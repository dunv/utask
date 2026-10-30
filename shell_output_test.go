package utask

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestShellSeparateOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	task, err := NewShellTask(WithShellCommand("/bin/sh", "-c", "printf out; printf err >&2"),
		WithShellStdout(&stdout), WithShellStderr(&stderr))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("output = %q / %q", stdout.String(), stderr.String())
	}
}

func TestShellFileOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	task, err := NewShellTask(WithShellCommand("/bin/sh", "-c", "printf out"), WithShellStdout(file))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
	// The caller's file remains open and owned by the caller.
	if _, err := file.WriteString("caller"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "outcaller" {
		t.Fatalf("output = %q, error = %v", data, err)
	}
}

// The type is comparable, but its interface field's value is not. Comparing
// writer interfaces must not panic, and both copy loops must serialize writes.
type indirectShellWriter struct {
	marker any
	target *bytes.Buffer
}

func (w indirectShellWriter) Write(p []byte) (int, error) { return w.target.Write(p) }

func TestShellNonComparableCombinedWriter(t *testing.T) {
	var output bytes.Buffer
	writer := indirectShellWriter{marker: []byte{1}, target: &output}
	task, err := NewShellTask(WithShellCommand("/bin/sh", "-c", "printf out; printf err >&2"),
		WithShellCombinedOutput(writer))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "outerr" && got != "errout" {
		t.Fatalf("combined output = %q", got)
	}
}

type failingShellWriter struct{ err error }

func (w failingShellWriter) Write([]byte) (int, error) { return 0, w.err }

func TestShellOutputError(t *testing.T) {
	want := errors.New("output destination failed")
	task, err := NewShellTask(WithShellCommand("/bin/sh", "-c", "printf out"),
		WithShellStdout(failingShellWriter{err: want}), WithShellWaitDelay(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Run(); !errors.Is(err, want) {
		t.Fatalf("error = %v, want output failure", err)
	}
}

func TestShellCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	task, err := NewShellTask(WithShellCommand("/bin/true"), WithShellContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Start(); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if task.(*shellTask).cmd != nil {
		t.Fatal("cancelled task acquired a process")
	}
}

func TestShellAlreadyGoneGroup(t *testing.T) {
	// Linux PIDs are bounded by pid_max (at most 2^22); this cannot identify
	// a real process group. Exercise ESRCH without risking an unrelated group.
	task := &shellTask{
		cmd:  &exec.Cmd{Process: &os.Process{Pid: 1<<31 - 1}},
		opts: options[shellTaskOptions]{specific: shellTaskOptions{shellTermSignal: syscall.SIGTERM}},
	}
	if err := task.terminateShellGroup(); err != nil {
		t.Fatalf("already-gone group was treated as a failure: %v", err)
	}
}
