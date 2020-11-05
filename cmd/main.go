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
	pid       = flag.Int("pid", 0, "the pid to hack")
	address   = flag.String("address", "127.0.0.1:4567", "address")
	typ       = flag.Int("type", 0, "type")

	duration = flag.Duration("duration", 10*time.Second, "")
	debug    = flag.Bool("debug", false, "display debug message")
)

func main() {
	flag.Parse()
	if *debug {
		logflags.Setup(true, "debugger", "")
	}
	if *duration <= 0 {
		*duration = 10 * time.Second
	}

	var (
		ppid int
		err  error
	)
	ctx := context.TODO()

	if *pid > 0 {
		ppid = *pid
	} else {
		if (*process) != "" {
			ppid, err = GetPidFromProcess(*process)
		} else {
			ppid, err = GetPidFromPod(ctx, *pod, *namespace)
		}
	}

	if err != nil {
		panic(err)
	}

	hacker, err := sql.NewSQLHacker(ppid)
	if err != nil {
		panic(err)
	}

	hacker.Invade(ctx, *duration)
}
