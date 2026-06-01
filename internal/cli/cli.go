package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/sqamsqam/setup/internal/devtools"
	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
)

var version = "dev"

type RunnerFactory func(dryRun bool) setupexec.CmdRunner

func SetVersion(v string) {
	version = v
}

func Version() string {
	return version
}

func BuildApp(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	if runnerFactory == nil {
		runnerFactory = defaultRunner
	}
	return &cli.Command{
		Name:    "setup",
		Usage:   "Provisioning helper for fresh Ubuntu 26.04 LXC containers",
		Version: version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Print actions without changing the system",
				Value: dryRun,
			},
		},
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {
			// Prevent urfave/cli from calling os.Exit(1) internally.
			// Error is printed by the caller (main.go).
		},
		Commands: []*cli.Command{
			bootstrapCmd(dryRun, runnerFactory),
			addUserCmd(dryRun, runnerFactory),
			installToolsCmd(dryRun, runnerFactory),
			devToolsCmd(dryRun, runnerFactory),
			fullCmd(dryRun, runnerFactory),
			versionCmd(),
		},
	}
}

func commandDryRun(cmd *cli.Command, fallback bool) bool {
	return fallback || cmd.Root().Bool("dry-run")
}

func defaultRunner(dryRun bool) setupexec.CmdRunner {
	if dryRun {
		return setupexec.NewDryRunner()
	}
	real := setupexec.NewRealRunner()
	real.Env = append(real.Env, "DEBIAN_FRONTEND=noninteractive")
	return real
}

func provisioningAction(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		if !isRoot() {
			fmt.Fprintln(os.Stderr, "WARNING: not running as root — provisioning may fail")
		}
		return action(ctx, cmd)
	}
}

func isRoot() bool {
	return os.Geteuid() == 0
}

func bootstrapCmd(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "bootstrap",
		Aliases: []string{"b"},
		Usage:   "Locale, system update, base packages, SSH hardening, unattended upgrades, Docker",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "timezone",
				Aliases: []string{"t"},
				Value:   "UTC",
				Usage:   "Timezone (default: UTC)",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			return system.Bootstrap(runnerFactory(commandDryRun(cmd, dryRun)), cmd.String("timezone"))
		}),
	}
}

func addUserCmd(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "add-user",
		Aliases: []string{"a"},
		Usage:   "Create sudo user with SSH key auth",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user",
				Aliases:  []string{"u"},
				Usage:    "Username for the new account",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "SSH public key content (visible in process list)",
			},
			&cli.StringFlag{
				Name:  "key-file",
				Usage: "Path to a file containing the SSH public key (safer)",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			username := cmd.String("user")
			pubkey := cmd.String("key")
			if keyFile := cmd.String("key-file"); keyFile != "" {
				if pubkey != "" {
					return fmt.Errorf("use either --key or --key-file, not both")
				}
				keyBytes, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("reading key file %s: %w", keyFile, err)
				}
				pubkey = strings.TrimSpace(string(keyBytes))
			}
			if pubkey == "" {
				return fmt.Errorf("either --key or --key-file is required")
			}
			return user.AddUser(runnerFactory(commandDryRun(cmd, dryRun)), username, pubkey)
		}),
	}
}

func installToolsCmd(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "install-tools",
		Aliases: []string{"i"},
		Usage:   "Install ripgrep, fd, bat, yq, glow, gh",
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			return tools.InstallAll(runnerFactory(commandDryRun(cmd, dryRun)))
		}),
	}
}

func devToolsCmd(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "devtools",
		Aliases: []string{"d"},
		Usage:   "Install Go (system-wide) and Node.js (per-user)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user",
				Aliases:  []string{"u"},
				Usage:    "Target user for Node.js installation",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Install both Go and Node.js",
			},
			&cli.BoolFlag{
				Name:  "go",
				Usage: "Install Go only",
			},
			&cli.BoolFlag{
				Name:  "node",
				Usage: "Install Node.js only",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			username := cmd.String("user")
			goOnly := cmd.Bool("go")
			nodeOnly := cmd.Bool("node")
			all := cmd.Bool("all")

			if all || (!goOnly && !nodeOnly) {
				goOnly = true
				nodeOnly = true
			}

			runner := runnerFactory(commandDryRun(cmd, dryRun))
			if goOnly {
				if err := devtools.InstallGo(runner); err != nil {
					return err
				}
			}
			if nodeOnly {
				if err := devtools.InstallNode(runner, username); err != nil {
					return err
				}
			}
			return nil
		}),
	}
}

func fullCmd(dryRun bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "full",
		Aliases: []string{"f"},
		Usage:   "Run all steps in sequence",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user",
				Aliases:  []string{"u"},
				Usage:    "Username for the new account",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "SSH public key content (visible in process list)",
			},
			&cli.StringFlag{
				Name:  "key-file",
				Usage: "Path to a file containing the SSH public key (safer)",
			},
			&cli.StringFlag{
				Name:    "timezone",
				Aliases: []string{"t"},
				Value:   "UTC",
				Usage:   "Timezone (default: UTC)",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			username := cmd.String("user")
			pubkey := cmd.String("key")
			if keyFile := cmd.String("key-file"); keyFile != "" {
				if pubkey != "" {
					return fmt.Errorf("use either --key or --key-file, not both")
				}
				keyBytes, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("reading key file %s: %w", keyFile, err)
				}
				pubkey = strings.TrimSpace(string(keyBytes))
			}
			if pubkey == "" {
				return fmt.Errorf("either --key or --key-file is required")
			}
			tz := cmd.String("timezone")
			runner := runnerFactory(commandDryRun(cmd, dryRun))

			setupexec.PrintStep("=== Full provisioning started ===")
			if err := system.Bootstrap(runner, tz); err != nil {
				return err
			}
			if err := user.AddUser(runner, username, pubkey); err != nil {
				return err
			}
			if err := tools.InstallAll(runner); err != nil {
				return err
			}
			if err := devtools.InstallAllDevTools(runner, username); err != nil {
				return err
			}
			setupexec.PrintDone("=== Full provisioning complete ===")
			return nil
		}),
	}
}

func versionCmd() *cli.Command {
	return &cli.Command{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print version info",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cmd.Root().Version)
			return nil
		},
	}
}
