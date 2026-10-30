package utask

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The test binary is also a real child process. Readiness is published only
// after signal handlers are installed, so cancellation never races startup.
func TestShellProcessHelper(t *testing.T) {
	ready := os.Getenv("UTASK_CHILD_READY")
	if ready == "" {
		return
	}
	if os.Getenv("UTASK_CHILD_RESIST") == "1" {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			for sig := range signals {
				if err := os.WriteFile(ready+".signal", []byte(sig.String()), 0600); err != nil {
					os.Exit(3)
				}
			}
		}()
	}
	if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
		os.Exit(2)
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestShellGroupCleanup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		exit     string
		cancel   bool
		resist   bool
		redirect bool
		signal   syscall.Signal
	}{
		{name: "cancel live shell and child", cancel: true, signal: syscall.SIGTERM},
		{name: "cancel kills resistant child without output pipe", cancel: true, resist: true, redirect: true, signal: syscall.SIGINT},
		{name: "exited shell with open pipe", exit: "exit 0", signal: syscall.SIGKILL},
		{name: "drain timeout escalates resistant child", exit: "exit 0", resist: true, signal: syscall.SIGTERM},
		{name: "nonzero exit with open pipe", exit: "exit 7", resist: true, signal: syscall.SIGTERM},
		{name: "nonzero exit without child output", exit: "exit 7", resist: true, redirect: true, signal: syscall.SIGTERM},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := startShellFixture(t, tc.exit, tc.resist, tc.redirect, tc.signal, 40*time.Millisecond)
			if tc.cancel {
				f.cancel()
			}
			err := f.wait(t)
			if tc.cancel {
				if err == nil {
					t.Fatal("cancelled task succeeded")
				}
			} else if tc.exit == "exit 7" {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
					t.Fatalf("error = %v, want exit status 7", err)
				}
			} else if !errors.Is(err, exec.ErrWaitDelay) {
				t.Fatalf("error = %v, want ErrWaitDelay", err)
			}
			awaitChildDead(t, f.child)
		})
	}
}

func TestShellSignalGrace(t *testing.T) {
	// The child's TERM trap proves that the configured signal arrives before
	// escalation. The child then ignores it, requiring group-wide SIGKILL.
	f := startShellFixture(t, "", true, true, syscall.SIGTERM, 300*time.Millisecond)
	started := time.Now()
	f.cancel()
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		data, _ := os.ReadFile(f.ready + ".signal")
		if string(data) == syscall.SIGTERM.String() {
			break
		}
		select {
		case err := <-f.done:
			f.finished = true
			t.Fatalf("task returned without delivering SIGTERM to child: %v", err)
		case <-tick.C:
		case <-deadline:
			t.Fatal("configured signal was not delivered to child")
		}
	}
	if childDead(f.child) {
		t.Fatal("resistant child killed before grace elapsed")
	}
	_ = f.wait(t)
	if elapsed := time.Since(started); elapsed < 300*time.Millisecond {
		t.Fatalf("cleanup returned after %s, before the configured grace interval", elapsed)
	}
	awaitChildDead(t, f.child)
}

func TestShellKillSkipsGrace(t *testing.T) {
	f := startShellFixture(t, "", false, true, syscall.SIGKILL, 3*time.Second)
	started := time.Now()
	f.cancel()
	_ = f.wait(t)
	if elapsed := time.Since(started); elapsed >= 2*time.Second {
		t.Fatalf("SIGKILL cleanup waited for a grace interval: %s", elapsed)
	}
	awaitChildDead(t, f.child)
}

func TestShellSuccessfulBackgroundChild(t *testing.T) {
	// A child with redirected output is not an output-drain failure. Preserve
	// successful daemon-launch behavior rather than killing every task group.
	f := startShellFixture(t, "exit 0", false, true, syscall.SIGKILL, 40*time.Millisecond)
	if err := f.wait(t); err != nil {
		t.Fatal(err)
	}
	if childDead(f.child) {
		t.Fatal("successful command's background child was killed")
	}
}

func TestShellZeroDelay(t *testing.T) {
	f := startShellFixture(t, "exit 0", false, false, syscall.SIGKILL, 0)
	// A zombie leader proves exit was observed without reaping. Holding its
	// identity is what makes subsequent group cancellation safe.
	awaitChildDead(t, f.leader)
	if _, err := os.Stat(fmt.Sprintf("/proc/%d/stat", f.leader)); err != nil {
		t.Fatalf("leader was reaped before output finished: %v", err)
	}
	select {
	case err := <-f.done:
		f.finished = true
		t.Fatalf("zero delay unexpectedly bounded output draining: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	// Cancellation must still reach the group after the shell has exited.
	f.cancel()
	if err := f.wait(t); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	awaitChildDead(t, f.child)
}

func TestShellLifecycleAndOutput(t *testing.T) {
	var output bytes.Buffer
	task, err := NewShellTask(WithShellCommand("/bin/sh", "-c", "printf a; printf b >&2; printf c"), WithShellCombinedOutput(&output))
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(); err == nil {
		t.Fatal("Wait before Start succeeded")
	}
	if err := task.Start(); err != nil {
		t.Fatal(err)
	}
	if err := task.Start(); err == nil {
		t.Fatal("duplicate Start succeeded")
	}
	if err := task.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(); err == nil {
		t.Fatal("duplicate Wait succeeded")
	}
	if output.String() != "abc" {
		t.Fatalf("combined output = %q, want abc", output.String())
	}
	if _, err := NewShellTask(WithShellCommand("true"), WithShellWaitDelay(-time.Second)); err == nil {
		t.Fatal("negative delay accepted")
	}
}

type shellFixture struct {
	cancel   context.CancelFunc
	done     chan error
	finished bool
	child    int
	leader   int
	ready    string
}

func startShellFixture(t *testing.T, exit string, resist, redirect bool, sig syscall.Signal, delay time.Duration) *shellFixture {
	t.Helper()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	ready := filepath.Join(dir, "ready")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := &shellFixture{cancel: cancel, done: make(chan error, 1), ready: ready}
	// This is installed before starting anything. On failure, cancel while the
	// task owns its unreaped leader; never signal a stale PGID after Wait.
	t.Cleanup(func() {
		cancel()
		if !f.finished {
			select {
			case <-f.done:
			case <-time.After(5 * time.Second):
				t.Error("fixture failed to stop")
			}
		}
		if f.child == 0 {
			data, _ := os.ReadFile(pidFile)
			f.child, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}
		if f.child > 1 && !childDead(f.child) {
			_ = syscall.Kill(f.child, syscall.SIGKILL)
			awaitChildDead(t, f.child)
		}
	})
	child := `"$UTASK_TEST_EXE" -test.run=^TestShellProcessHelper$`
	if redirect {
		child += " >/dev/null 2>&1"
	}
	body := child + " &\nchild=$!\nprintf '%s' \"$child\" >\"$UTASK_PID_FILE\"\n" +
		"while [ ! -f \"$UTASK_CHILD_READY\" ]; do sleep 0.01; done\n"
	if exit == "" {
		body += "wait \"$child\"\n"
	} else {
		body += exit + "\n"
	}
	var output bytes.Buffer
	task, err := NewShellTask(
		WithShellCommand("/bin/sh", "-c", body), WithShellContext(ctx),
		WithShellTermSignal(sig), WithShellWaitDelay(delay), WithShellCombinedOutput(&output),
		WithShellEnvironment(append(os.Environ(), "UTASK_TEST_EXE="+executable,
			"UTASK_CHILD_READY="+ready, "UTASK_PID_FILE="+pidFile, "UTASK_CHILD_RESIST="+map[bool]string{true: "1", false: "0"}[resist])),
	)
	if err != nil {
		f.finished = true
		t.Fatal(err)
	}
	if err := task.Start(); err != nil {
		f.finished = true
		t.Fatal(err)
	}
	f.leader = task.(*shellTask).cmd.Process.Pid
	go func() { f.done <- task.Wait() }()
	deadline := time.After(5 * time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		if _, err := os.Stat(ready); err == nil {
			data, err := os.ReadFile(pidFile)
			if err == nil {
				f.child, err = strconv.Atoi(strings.TrimSpace(string(data)))
				if err == nil && f.child > 1 {
					return f
				}
			}
		}
		select {
		case <-tick.C:
		case <-deadline:
			t.Fatal("child did not become ready")
		}
	}
}

func (f *shellFixture) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-f.done:
		f.finished = true
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("task did not return within bound")
		return nil
	}
}

func childDead(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	// Orphaned zombies are dead but may await PID 1's reaping.
	_, state, ok := strings.Cut(string(data), ") ")
	return ok && strings.HasPrefix(state, "Z ")
}

func awaitChildDead(t *testing.T, pid int) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for !childDead(pid) {
		select {
		case <-tick.C:
		case <-deadline:
			t.Fatalf("child %d survived group cleanup", pid)
		}
	}
}
