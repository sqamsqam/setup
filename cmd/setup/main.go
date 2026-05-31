package main

import (
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
			a == "devtools" || a == "full" || a == "version" ||
			a == "--help" || a == "-h" {
			isCLI = true
			break
		}
	}

	if !isCLI {
		if !isRoot() {
			_, _ = os.Stderr.WriteString("WARNING: not running as root — provisioning may fail\n")
		}
		tui.Run(dryRun)
		return
	}

	isVersion := false
	for _, a := range args {
		if a == "--dry-run" {
			continue
		}
		if a == "version" || a == "--help" || a == "-h" {
			isVersion = true
			break
		}
	}

	cli.Run(args)

	if !isRoot() && !isVersion {
		_, _ = os.Stderr.WriteString("WARNING: not running as root — provisioning may fail\n")
	}
}

func isRoot() bool {
	return os.Geteuid() == 0
}
