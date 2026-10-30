[![Go package](https://github.com/dunv/utask/actions/workflows/go.yaml/badge.svg)](https://github.com/dunv/utask/actions/workflows/go.yaml)

## Shell task lifecycle (Linux)

A shell task runs the configured executable directly; use `/bin/sh -c` explicitly
when shell syntax is needed. Each task has its own process group.

`Start` begins execution and lifecycle supervision, including cancellation even
before `Wait` is called. Call `Wait` once to collect the result, or use `Run`.
Starting an already-started task or waiting twice is an error.

Group cleanup is triggered by:

- Context cancellation: signal the group, not just the executable.
- An unsuccessful executable exit: signal any remaining group members, including
  children that have redirected their output.
- Output-copy failure or output-drain timeout after executable exit: signal any
  remaining group members, even though the group leader has exited.

`WithShellTermSignal` selects the initial cleanup signal (default `SIGTERM`).
`WithShellWaitDelay` selects both the termination grace interval and the
output-drain timeout (default one second). With a positive delay, cleanup waits
that grace interval and then sends **group-wide `SIGKILL`**, even if the leader
has exited or the output pipes have closed. An initial `SIGKILL` skips the grace
interval. Cleanup waits the full grace interval for other signals, including
when no live children remain.

Draining and termination are separate phases. Output draining can take one delay
interval after executable exit before triggering cleanup; graceful cleanup can
then take another interval, followed by at most one final drain interval. There
is no claim that Go's `exec.Cmd.WaitDelay` kills a process group. A zero delay
explicitly disables both drain timeouts and forced escalation; an uncooperative
child may then survive the initial signal or keep the task waiting indefinitely.
Negative delays are rejected.

The executable's nonzero exit error takes precedence over cancellation and
output errors. If it exits successfully, cancellation takes precedence over
output errors; a drain timeout is reported as `exec.ErrWaitDelay`. Unexpected
cleanup errors are joined with the command error so `errors.Is` and `errors.As`
remain usable. An already-gone group is not a cleanup failure.

Successful commands whose output finishes normally do not trigger group cleanup.
This preserves successful background launches with redirected output. Stdout and
stderr sent to the same comparable writer share one pipe to preserve ordering;
caller-supplied files remain caller-owned.

### Identity and limits

The Linux implementation observes executable exit with `waitid(WNOWAIT)` and
keeps the leader unreaped until all possible group signals are finished. This
reserves its numeric PID/PGID during draining and escalation, avoiding a signal
to an unrelated group that reused the number after reaping. The task must
exclusively own reaping its child: do not independently reap it with a host-wide
`waitpid(-1)` handler or configure automatic child reaping.

Cleanup covers processes that remain in the task's original group and that the
caller is permitted to signal. It cannot capture descendants deliberately leaving
the group, force a process out of uninterruptible kernel I/O, or forcibly unblock
a caller-provided output writer stuck inside `Write`. Those cases are not covered
by the return-time bounds. Dead orphaned children may remain zombies until their
new parent reaps them; zombies cannot execute commands or retain output pipes.

## Verification

The upstream workflow uses Go 1.21 on Linux. Run its checks with the existing
manual Ansible test disabled, and include the race detector for lifecycle changes:

```sh
go build ./...
CI=true go test ./...
CI=true go test -race ./...
go vet ./...
```

The subprocess regressions check descendant death as well as bounded return,
including exited leaders, signal-resistant children and redirected output.
