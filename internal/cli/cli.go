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

func Run(args []string) {
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	if args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return
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

	// Check for help flags in remaining args
	for _, a := range remaining {
		if a == "--help" || a == "-h" {
			printCommandHelp(cmd)
			return
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

func printUsage() {
	fmt.Println("Usage: setup [--dry-run] <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bootstrap               Locale, system update, base packages, SSH hardening,")
	fmt.Println("                          unattended upgrades, Docker")
	fmt.Println("                          Options: --timezone <zone> (default: Australia/Sydney)")
	fmt.Println("  add-user                Create sudo user with SSH key auth")
	fmt.Println("                          Options: --user <name> --key \"<ssh-public-key>\"")
	fmt.Println("  install-tools           Install ripgrep, fd, bat, yq, glow, gh")
	fmt.Println("  devtools                Install Go (system-wide) and Node.js (per-user)")
	fmt.Println("                          Options: --user <name> [--all] [--go] [--node]")
	fmt.Println("  full                    Run all steps in sequence")
	fmt.Println("                          Options: --user <name> --key \"<pubkey>\"")
	fmt.Println("                                   [--timezone <zone>]")
	fmt.Println("  version                 Print version info")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  --dry-run               Log commands without executing")
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "bootstrap":
		fmt.Println("Usage: setup bootstrap [--timezone <zone>]")
		fmt.Println()
		fmt.Println("Run root-level system bootstrap including locale generation, apt update")
		fmt.Println("and upgrade, base package installation, SSH hardening, unattended")
		fmt.Println("security upgrades, timezone configuration, and Docker installation.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --timezone <zone>  Timezone (default: Australia/Sydney)")
	case "add-user":
		fmt.Println("Usage: setup add-user --user <name> --key \"<ssh-public-key>\"")
		fmt.Println()
		fmt.Println("Create a sudo user with SSH key authentication. The user is granted")
		fmt.Println("passwordless sudo, has linger enabled, and is added to the SSH")
		fmt.Println("AllowUsers list.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --user <name>      Username for the new account")
		fmt.Println("  --key \"<key>\"     SSH public key content")
	case "install-tools":
		fmt.Println("Usage: setup install-tools")
		fmt.Println()
		fmt.Println("Install CLI tools: ripgrep, fd, bat, yq, glow, gh.")
		fmt.Println("Downloads from official sources and GitHub releases.")
	case "devtools":
		fmt.Println("Usage: setup devtools --user <name> [--all] [--go] [--node]")
		fmt.Println()
		fmt.Println("Install development toolchains. Go is installed system-wide from")
		fmt.Println("go.dev (SHA256 verified). Node.js is installed per-user via fnm.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --user <name>  Target user for Node.js installation")
		fmt.Println("  --all          Install both Go and Node.js")
		fmt.Println("  --go           Install Go only")
		fmt.Println("  --node         Install Node.js only")
	case "full":
		fmt.Println("Usage: setup full --user <name> --key \"<pubkey>\" [--timezone <zone>]")
		fmt.Println()
		fmt.Println("Run the entire provisioning flow: bootstrap, add-user, install-tools,")
		fmt.Println("and devtools in sequence.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --user <name>      Username for the new account")
		fmt.Println("  --key \"<pubkey>\"  SSH public key content")
		fmt.Println("  --timezone <zone>  Timezone (default: Australia/Sydney)")
	default:
		printUsage()
	}
}

func runBootstrap(runner setupexec.CmdRunner, args []string) {
	tz := setupexec.DefaultTimezone
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--timezone" && i+1 < len(args):
			tz = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--timezone="):
			tz = strings.TrimPrefix(args[i], "--timezone=")
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				os.Exit(1)
			}
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
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				os.Exit(1)
			}
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
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				os.Exit(1)
			}
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
	tz := setupexec.DefaultTimezone
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
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				os.Exit(1)
			}
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
