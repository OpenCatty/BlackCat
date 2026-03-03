//go:build ignore

// basic_task.go shows the minimal usage of the blackcat agent:
// start opencode serve, run a task, print the session ID.
//
// Run with:
//
//	go run examples/basic_task.go "Add unit tests to the calculator package"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/startower-observability/blackcat/agent"
	"github.com/startower-observability/blackcat/opencode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run examples/basic_task.go <prompt>")
		os.Exit(1)
	}
	prompt := os.Args[1]

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Start opencode serve as a supervised child process.
	sup := opencode.NewSupervisor(opencode.SupervisorConfig{
		Port: 4096,
	})
	if err := sup.Start(ctx); err != nil {
		log.Fatalf("start opencode: %v", err)
	}
	defer sup.Stop()
	fmt.Fprintf(os.Stderr, "opencode running at %s\n", sup.BaseURL())

	// 2. Create the agent pointing at the running server.
	ag := agent.New(agent.Config{
		OpenCodeAddr: sup.BaseURL(),
		AutoPermit:   false,
		Verbose:      true,
		Output:       os.Stderr,
	})

	// 3. Run the task.
	result, err := ag.Run(ctx, opencode.TaskRequest{
		Prompt: prompt,
	})
	if err != nil {
		log.Fatalf("task failed: %v", err)
	}

	fmt.Printf("Done.\nSession ID : %s\nMessages   : %d\n", result.SessionID, len(result.Messages))
}
