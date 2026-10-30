package utask

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// Linux SIGCHLD siginfo_t layout: three header words (errno/code ordering
// varies on MIPS), a pointer-aligned union, then pid, uid and exit status.
// Only WEXITED events are requested: status zero means a successful exit;
// signal deaths have a nonzero signal number. Keep the full kernel buffer.
type shellExitInfo struct {
	header [3]int32
	_      [unsafe.Sizeof(uintptr(0)) / 8]int32
	pid    int32
	uid    uint32
	status int32
	_      [128 - 24 - unsafe.Sizeof(uintptr(0))/8*4]byte
}

type shellExit struct {
	success bool
	err     error
}

func observeShellExit(pid int) shellExit {
	var info shellExitInfo
	for {
		_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, 1 /* P_PID */, uintptr(pid),
			uintptr(unsafe.Pointer(&info)), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return shellExit{err: fmt.Errorf("utask: observe shell exit: %w", errno)}
		}
		return shellExit{success: info.status == 0}
	}
}

// waitShell exclusively owns reaping. WNOWAIT keeps the leader's PID reserved
// even after exit, so all group signals precede cmd.Wait and cannot hit a reused
// PGID. A host must not independently reap this task's child (e.g. waitpid(-1)).
func (t *shellTask) waitShell(output *shellOutput) error {
	defer output.close()
	exited := make(chan shellExit, 1)
	go func() { exited <- observeShellExit(t.cmd.Process.Pid) }()

	var exit shellExit
	var cancelErr, outputErr, cleanupErr error
	cleaned := false
	cleanup := func() {
		cleaned = true
		cleanupErr = t.terminateShellGroup()
	}
	select {
	case exit = <-exited:
	case <-t.opts.ctx.Done():
		cancelErr = t.opts.ctx.Err()
		cleanup()
		exit = <-exited
	}

	// If exit observation fails, ownership cannot be established. Do not add
	// post-reap group signals; report the failure and finish releasing resources.
	if exit.err != nil {
		output.close()
		<-output.done
		return errors.Join(exit.err, cleanupErr, t.cmd.Wait())
	}
	if !exit.success && !cleaned {
		cleanup()
	}

	drained := false
	if !cleaned {
		timer, expiry := shellTimer(t.opts.specific.shellWaitDelay)
		select {
		case outputErr = <-output.done:
			drained = true
			if outputErr != nil {
				cleanup()
			}
		case <-t.opts.ctx.Done():
			cancelErr = t.opts.ctx.Err()
			cleanup()
		case <-expiry:
			outputErr = exec.ErrWaitDelay
			cleanup()
		}
		if timer != nil {
			timer.Stop()
		}
	}
	if !drained {
		timer, expiry := shellTimer(t.opts.specific.shellWaitDelay)
		select {
		case err := <-output.done:
			if outputErr == nil {
				outputErr = err
			}
		case <-expiry:
			output.close()
			<-output.done
			if outputErr == nil {
				outputErr = exec.ErrWaitDelay
			}
		}
		if timer != nil {
			timer.Stop()
		}
	}

	// Preserve os/exec's precedence and errors.As(*exec.ExitError) behavior.
	err := t.cmd.Wait()
	if err == nil {
		err = cancelErr
	}
	if err == nil {
		err = outputErr
	}
	if cleanupErr != nil {
		return errors.Join(err, cleanupErr)
	}
	return err
}

func (t *shellTask) terminateShellGroup() error {
	pid := t.cmd.Process.Pid
	if pid <= 1 {
		return fmt.Errorf("utask: refusing to signal invalid process group %d", pid)
	}
	signal := func(sig syscall.Signal) error {
		err := syscall.Kill(-pid, sig)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("utask: signal group %d with %s: %w", pid, sig, err)
		}
		return nil
	}
	err := signal(t.opts.specific.shellTermSignal)
	if t.opts.specific.shellTermSignal == syscall.SIGKILL || t.opts.specific.shellWaitDelay == 0 {
		return err
	}
	// The unreaped leader also keeps an otherwise empty group observable.
	// Wait the grace interval rather than interpreting kill(pgid, 0) as proof
	// that living descendants remain, then escalate before releasing identity.
	time.Sleep(t.opts.specific.shellWaitDelay)
	killErr := signal(syscall.SIGKILL)
	if err != nil || killErr != nil {
		return errors.Join(err, killErr)
	}
	return nil
}

func shellTimer(delay time.Duration) (*time.Timer, <-chan time.Time) {
	if delay == 0 {
		return nil, nil
	}
	timer := time.NewTimer(delay)
	return timer, timer.C
}
