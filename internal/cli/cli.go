package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/sqamsqam/setup/internal/devtools"
	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
)

const defaultTimezone = "Australia/Sydney"

func Run(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: setup <command> [options]\n")
		fmt.Fprintf(os.Stderr, "Commands: bootstrap, add-user, install-tools, devtools, full, version\n")
		os.Exit(1)
	}

	dryRun := false
	cmd := args[0]
	remaining := args[1:]

	if cmd == "--dry-run" {
		dryRun = true
		if len(remaining) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: setup --dry-run <command> [options]\n")
			os.Exit(1)
		}
		cmd = remaining[0]
		remaining = remaining[1:]
	}

	for _, a := range remaining {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	var runner setupexec.CmdRunner
	if dryRun {
		runner = setupexec.NewDryRunner()
	} else {
		real := setupexec.NewRealRunner()
		real.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		runner = real
	}

	switch cmd {
	case "bootstrap":
		runBootstrap(runner, remaining)
	case "add-user":
		runAddUser(runner, remaining)
	case "install-tools":
		runInstallTools(runner)
	case "devtools":
		runDevTools(runner, remaining)
	case "full":
		runFull(runner, remaining)
	case "version":
		fmt.Println(Version())
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Fprintf(os.Stderr, "Commands: bootstrap, add-user, install-tools, devtools, full, version\n")
		os.Exit(1)
	}
}

func runBootstrap(runner setupexec.CmdRunner, args []string) {
	tz := defaultTimezone
	for _, a := range args {
		if strings.HasPrefix(a, "--timezone=") {
			tz = strings.TrimPrefix(a, "--timezone=")
		}
	}

	if err := system.Bootstrap(runner, tz); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func runAddUser(runner setupexec.CmdRunner, args []string) {
	var username, pubkey string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--user" && i+1 < len(args):
			username = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--user="):
			username = strings.TrimPrefix(args[i], "--user=")
		case args[i] == "--key" && i+1 < len(args):
			pubkey = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--key="):
			pubkey = strings.TrimPrefix(args[i], "--key=")
		}
	}

	if username == "" || pubkey == "" {
		fmt.Fprintf(os.Stderr, "Usage: setup add-user --user <username> --key \"<ssh-public-key>\"\n")
		os.Exit(1)
	}

	if err := user.AddUser(runner, username, pubkey); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func runInstallTools(runner setupexec.CmdRunner) {
	if err := tools.InstallAll(runner); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func runDevTools(runner setupexec.CmdRunner, args []string) {
	var username string
	all := false
	goOnly := false
	nodeOnly := false

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--user" && i+1 < len(args):
			username = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--user="):
			username = strings.TrimPrefix(args[i], "--user=")
		case args[i] == "--all":
			all = true
		case args[i] == "--go":
			goOnly = true
		case args[i] == "--node":
			nodeOnly = true
		}
	}

	if username == "" {
		fmt.Fprintf(os.Stderr, "Usage: setup devtools --user <username> [--all] [--go] [--node]\n")
		os.Exit(1)
	}

	if all || (!goOnly && !nodeOnly) {
		goOnly = true
		nodeOnly = true
	}

	var err error
	if goOnly {
		err = devtools.InstallGo(runner)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}
	if nodeOnly {
		err = devtools.InstallNode(runner, username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}
}

func runFull(runner setupexec.CmdRunner, args []string) {
	tz := defaultTimezone
	var username, pubkey string

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--user" && i+1 < len(args):
			username = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--user="):
			username = strings.TrimPrefix(args[i], "--user=")
		case args[i] == "--key" && i+1 < len(args):
			pubkey = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--key="):
			pubkey = strings.TrimPrefix(args[i], "--key=")
		case strings.HasPrefix(args[i], "--timezone="):
			tz = strings.TrimPrefix(args[i], "--timezone=")
		case args[i] == "--timezone" && i+1 < len(args):
			tz = args[i+1]
			i++
		}
	}

	if username == "" || pubkey == "" {
		fmt.Fprintf(os.Stderr, "Usage: setup full --user <username> --key \"<ssh-public-key>\" [--timezone Tz]\n")
		os.Exit(1)
	}

	setupexec.PrintStep("=== Full provisioning started ===")

	if err := system.Bootstrap(runner, tz); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if err := user.AddUser(runner, username, pubkey); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if err := tools.InstallAll(runner); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if err := devtools.InstallAllDevTools(runner, username); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	setupexec.PrintDone("=== Full provisioning complete ===")
}

var version = "dev"

func SetVersion(v string) {
	version = v
}

func Version() string {
	return version
}
