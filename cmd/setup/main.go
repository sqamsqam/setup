package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sqamsqam/setup/internal/cli"
	"github.com/sqamsqam/setup/internal/tui"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cli.SetVersion(version + " (" + commit + ", " + buildDate + ")")

	args := os.Args[1:]

	dryRun := false
	cleanArgs := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--dry-run", "--dry-run=true":
			dryRun = true
		case "--dry-run=false":
			dryRun = false
		default:
			cleanArgs = append(cleanArgs, a)
		}
	}

	if isTUI(cleanArgs) {
		if !isRoot() {
			_, _ = os.Stderr.WriteString("WARNING: not running as root — provisioning may fail\n")
		}
		tui.Run(dryRun)
		return
	}

	app := cli.BuildApp(dryRun, nil)
	if err := app.Run(context.Background(), append([]string{os.Args[0]}, cleanArgs...)); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func isTUI(args []string) bool {
	return len(args) == 0
}

func isRoot() bool {
	return os.Geteuid() == 0
}
