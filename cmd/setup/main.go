package main

import (
	"os"
	"strings"

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
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	isCLI := false
	for _, a := range args {
		if a == "--dry-run" {
			continue
		}
		if a == "bootstrap" || a == "add-user" || a == "install-tools" ||
			a == "devtools" || a == "full" || a == "version" {
			isCLI = true
			break
		}
	}

	if !isCLI {
		if !isRoot() {
			os.Stderr.WriteString("WARNING: not running as root — provisioning may fail\n")
		}
		tui.Run(dryRun)
		return
	}

	cli.Run(args)

	if !isRoot() && !strings.HasPrefix(strings.Join(args, " "), "version") {
		os.Stderr.WriteString("WARNING: not running as root — provisioning may fail\n")
	}
}

func isRoot() bool {
	return os.Geteuid() == 0
}
