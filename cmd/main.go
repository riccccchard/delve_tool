package main

import (
	"context"
	"delve_tool/chaos/sql"
	"flag"
	"time"

	"github.com/go-delve/delve/pkg/logflags"
)

var (
	pod       = flag.String("pod", "", "the process to hack")
	namespace = flag.String("namespace", "", "the namespace of the pod")
	process   = flag.String("process", "", "the process to hack")

	timeout = flag.Int("timeout", 10, "")
	debug   = flag.Bool("debug", false, "display debug message")
)

func main() {
	flag.Parse()
	if *debug {
		logflags.Setup(true, "debugger", "")
	}
	if *timeout <= 0 {
		*timeout = 10
	}

	var (
		pid int
		err error
	)
	ctx := context.TODO()

	if (*process) != "" {
		pid, err = GetPidFromProcess(*process)
	} else {
		pid, err = GetPidFromPod(ctx, *pod, *namespace)
	}
	if err != nil {
		panic(err)
	}

	hacker, err := sql.NewSQLHacker(pid)
	if err != nil {
		panic(err)
	}

	period := time.Duration(*timeout) * time.Second
	hacker.Invade(ctx, period)
}
