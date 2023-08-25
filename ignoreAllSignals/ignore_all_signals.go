package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	signal.Ignore(syscall.SIGINT)
	signal.Ignore(syscall.SIGTERM)

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel,
		syscall.SIGABRT,
		syscall.SIGALRM,
		syscall.SIGBUS,
		syscall.SIGCHLD,
		syscall.SIGCONT,
		syscall.SIGFPE,
		syscall.SIGHUP,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGIO,
		syscall.SIGIOT,
		syscall.SIGPIPE,
		syscall.SIGPROF,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGSYS,
		syscall.SIGTERM,
		syscall.SIGTRAP,
		syscall.SIGTSTP,
		syscall.SIGTTIN,
		syscall.SIGTTOU,
		syscall.SIGURG,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGVTALRM,
		syscall.SIGWINCH,
		syscall.SIGXCPU,
		syscall.SIGXFSZ,
	)

	for {
		select {
		case sig := <-signalChannel:
			fmt.Println("trapped", sig)
		case <-time.After(50 * time.Millisecond):
			fmt.Println("still running")
		}
	}

}
