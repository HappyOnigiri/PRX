package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HappyOnigiri/PRX/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		jsonMode := false
		for _, arg := range os.Args[1:] {
			if arg == "--json" {
				jsonMode = true
				break
			}
		}
		if jsonMode {
			cli.PrintError(os.Stdout, err)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}
