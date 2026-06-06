package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/sqamsqam/setup/internal/devtools"
	"github.com/sqamsqam/setup/internal/diagnostics"
	dockermaint "github.com/sqamsqam/setup/internal/docker"
	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/firewall"
	sysgroup "github.com/sqamsqam/setup/internal/group"
	"github.com/sqamsqam/setup/internal/security"
	"github.com/sqamsqam/setup/internal/service"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/updates"
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
	return BuildAppWithMode(dryRun, false, runnerFactory)
}

func BuildAppWithMode(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:    "setup",
		Usage:   "Get a fresh Ubuntu 26.04 LXC ready to use",
		Version: version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Print actions without changing the system",
				Value: dryRun,
			},
			&cli.BoolFlag{
				Name:  "demo",
				Usage: "Preview actions without changing the system or showing dry-run labels",
				Value: demo,
			},
		},
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {
			// Prevent urfave/cli from calling os.Exit(1) internally.
			// Error is printed by the caller (main.go).
		},
		Commands: []*cli.Command{
			bootstrapCmd(dryRun, demo, runnerFactory),
			addUserCmd(dryRun, demo, runnerFactory),
			installToolsCmd(dryRun, demo, runnerFactory),
			devToolsCmd(dryRun, demo, runnerFactory),
			doctorCmd(dryRun, demo, runnerFactory),
			firewallCmd(dryRun, demo, runnerFactory),
			fail2banCmd(dryRun, demo, runnerFactory),
			dockerCmd(dryRun, demo, runnerFactory),
			updatesCmd(dryRun, demo, runnerFactory),
			serviceCmd(dryRun, demo, runnerFactory),
			groupCmd(dryRun, demo, runnerFactory),
			fullCmd(dryRun, demo, runnerFactory),
			versionCmd(),
		},
	}
}

func commandDryRun(cmd *cli.Command, fallback bool) bool {
	return fallback || cmd.Root().Bool("dry-run") || cmd.Root().Bool("demo")
}

func commandDemo(cmd *cli.Command, fallback bool) bool {
	return fallback || cmd.Root().Bool("demo")
}

func commandRunner(cmd *cli.Command, dryRun, demo bool, runnerFactory RunnerFactory) setupexec.CmdRunner {
	effectiveDryRun := commandDryRun(cmd, dryRun)
	effectiveDemo := commandDemo(cmd, demo)
	if runnerFactory != nil {
		return runnerFactory(effectiveDryRun)
	}
	return defaultRunnerForMode(effectiveDryRun, effectiveDemo)
}

func defaultRunner(dryRun bool) setupexec.CmdRunner {
	return defaultRunnerForMode(dryRun, false)
}

func defaultRunnerForMode(dryRun, demo bool) setupexec.CmdRunner {
	if demo {
		return setupexec.NewDemoRunner()
	}
	if dryRun {
		return setupexec.NewDryRunner()
	}
	real := setupexec.NewRealRunner()
	real.Env = append(real.Env, "DEBIAN_FRONTEND=noninteractive")
	return real
}

func provisioningAction(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		if !cmd.Root().Bool("demo") && !isRoot() {
			fmt.Fprintln(os.Stderr, "WARNING: not running as root — provisioning may fail")
		}
		return action(ctx, cmd)
	}
}

func isRoot() bool {
	return os.Geteuid() == 0
}

func bootstrapCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "base",
		Usage: "Prepare locale, packages, SSH hardening, unattended upgrades, and Docker",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "timezone",
				Aliases: []string{"t"},
				Value:   "UTC",
				Usage:   "Timezone (default: UTC)",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			return system.Bootstrap(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("timezone"))
		}),
	}
}

func addUserCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage login and setup-owned service users",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "user",
				Aliases: []string{"u"},
				Usage:   "Username for the compatibility login-user setup",
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
		Commands: userCommands(dryRun, demo, runnerFactory),
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			username := cmd.String("user")
			if username == "" {
				return fmt.Errorf("missing required flag: --user")
			}
			pubkey, err := keyFromFlags(cmd, true)
			if err != nil {
				return err
			}
			return user.AddUser(commandRunner(cmd, dryRun, demo, runnerFactory), username, pubkey)
		}),
	}
}

func userCommands(dryRun, demo bool, runnerFactory RunnerFactory) []*cli.Command {
	userFlag := func() cli.Flag {
		return &cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "Target username", Required: true}
	}
	groupFlag := func() cli.Flag {
		return &cli.StringFlag{Name: "group", Usage: "Existing group name"}
	}
	groupsFlag := func() cli.Flag {
		return &cli.StringSliceFlag{Name: "group", Usage: "Existing group name; may be repeated"}
	}
	keyFlag := func() cli.Flag {
		return &cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: "SSH public key content (visible in process list)"}
	}
	keyFileFlag := func() cli.Flag {
		return &cli.StringFlag{Name: "key-file", Usage: "Path to a file containing the SSH public key (safer)"}
	}

	return []*cli.Command{
		{
			Name:  "create",
			Usage: "Create or reuse a login user and apply selected access actions",
			Flags: []cli.Flag{
				userFlag(),
				keyFlag(),
				keyFileFlag(),
				&cli.BoolFlag{Name: "allow-ssh", Usage: "Add the user to setup-managed SSH AllowUsers"},
				&cli.BoolFlag{Name: "sudo", Usage: "Enable setup-managed passwordless sudo"},
				&cli.BoolFlag{Name: "linger", Usage: "Enable systemd user lingering"},
				groupsFlag(),
			},
			Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
				pubkey, err := keyFromFlags(cmd, false)
				if err != nil {
					return err
				}
				return user.CreateLoginUserSelected(
					commandRunner(cmd, dryRun, demo, runnerFactory),
					cmd.String("user"),
					pubkey,
					cmd.Bool("allow-ssh"),
					cmd.Bool("sudo"),
					cmd.Bool("linger"),
					cmd.StringSlice("group"),
				)
			}),
		},
		{
			Name:  "service",
			Usage: "Manage setup-owned no-login service users",
			Commands: []*cli.Command{
				{
					Name:  "create",
					Usage: "Create a setup-owned system no-login user under /var/lib/<user>",
					Flags: []cli.Flag{userFlag(), groupsFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.CreateServiceUser(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.StringSlice("group"))
					}),
				},
			},
		},
		{
			Name:  "ssh",
			Usage: "Manage setup-owned SSH key and AllowUsers access",
			Commands: []*cli.Command{
				{
					Name:  "key",
					Usage: "Manage authorized SSH keys",
					Commands: []*cli.Command{
						{
							Name:  "add",
							Usage: "Add an authorized SSH public key idempotently",
							Flags: []cli.Flag{userFlag(), keyFlag(), keyFileFlag()},
							Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
								pubkey, err := keyFromFlags(cmd, true)
								if err != nil {
									return err
								}
								return user.AddAuthorizedKey(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), pubkey)
							}),
						},
					},
				},
				{
					Name:  "allow",
					Usage: "Add the user to setup-managed SSH AllowUsers",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.AllowSSH(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
				{
					Name:  "deny",
					Usage: "Remove the user from setup-managed SSH AllowUsers",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.DenySSH(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
			},
		},
		{
			Name:  "sudo",
			Usage: "Manage setup-owned passwordless sudo",
			Commands: []*cli.Command{
				{
					Name:  "enable",
					Usage: "Enable setup-managed passwordless sudo",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.EnablePasswordlessSudo(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
				{
					Name:  "disable",
					Usage: "Remove setup-managed passwordless sudo",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.DisablePasswordlessSudo(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
			},
		},
		{
			Name:  "linger",
			Usage: "Manage systemd user lingering",
			Commands: []*cli.Command{
				{
					Name:  "enable",
					Usage: "Enable lingering",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.EnableLinger(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
				{
					Name:  "disable",
					Usage: "Disable lingering",
					Flags: []cli.Flag{userFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						return user.DisableLinger(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					}),
				},
			},
		},
		{
			Name:  "group",
			Usage: "Manage membership in existing groups",
			Commands: []*cli.Command{
				{
					Name:  "add",
					Usage: "Add the user to an existing group",
					Flags: []cli.Flag{userFlag(), groupFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						group := cmd.String("group")
						if group == "" {
							return fmt.Errorf("missing required flag: --group")
						}
						return user.AddGroup(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), group)
					}),
				},
				{
					Name:  "remove",
					Usage: "Remove the user from an existing group",
					Flags: []cli.Flag{userFlag(), groupFlag()},
					Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
						group := cmd.String("group")
						if group == "" {
							return fmt.Errorf("missing required flag: --group")
						}
						return user.RemoveGroup(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), group)
					}),
				},
			},
		},
		{
			Name:  "disable",
			Usage: "Lock access without deleting user data",
			Flags: []cli.Flag{userFlag()},
			Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
				return user.DisableUser(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
			}),
		},
		{
			Name:  "delete",
			Usage: "Delete an account after disabling access",
			Flags: []cli.Flag{
				userFlag(),
				&cli.BoolFlag{Name: "remove-home", Usage: "Also remove the user's home directory"},
			},
			Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
				return user.DeleteUser(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.Bool("remove-home"))
			}),
		},
	}
}

func keyFromFlags(cmd *cli.Command, required bool) (string, error) {
	pubkey := cmd.String("key")
	if keyFile := cmd.String("key-file"); keyFile != "" {
		if pubkey != "" {
			return "", fmt.Errorf("use either --key or --key-file, not both")
		}
		keyBytes, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("reading key file %s: %w", keyFile, err)
		}
		pubkey = strings.TrimSpace(string(keyBytes))
	}
	if required && pubkey == "" {
		return "", fmt.Errorf("either --key or --key-file is required")
	}
	return strings.TrimSpace(pubkey), nil
}

func groupCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	groupFlag := func(required bool) cli.Flag {
		return &cli.StringFlag{Name: "group", Usage: "Group name", Required: required}
	}
	userFlag := func() cli.Flag {
		return &cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "Target username", Required: true}
	}

	return &cli.Command{
		Name:  "group",
		Usage: "Manage system groups and group membership",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a system group if needed",
				Flags: []cli.Flag{groupFlag(true)},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return sysgroup.Create(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("group"))
				}),
			},
			{
				Name:  "delete",
				Usage: "Delete a system group after safety checks",
				Flags: []cli.Flag{
					groupFlag(true),
					&cli.BoolFlag{Name: "yes", Usage: "Confirm group deletion"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("group delete requires --yes")
					}
					return sysgroup.Delete(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("group"))
				}),
			},
			{
				Name:  "list",
				Usage: "List system groups",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					groups, err := sysgroup.List(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					if len(groups) == 0 {
						fmt.Println("No groups found.")
						return nil
					}
					fmt.Println(strings.Join(groups, "\n"))
					return nil
				}),
			},
			{
				Name:  "user",
				Usage: "Manage user membership in groups",
				Commands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a user to an existing group",
						Flags: []cli.Flag{userFlag(), groupFlag(true)},
						Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
							return sysgroup.AddUser(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("group"))
						}),
					},
					{
						Name:  "remove",
						Usage: "Remove a user from a group",
						Flags: []cli.Flag{userFlag(), groupFlag(true)},
						Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
							return sysgroup.RemoveUser(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("group"))
						}),
					},
				},
			},
		},
	}
}

func installToolsCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "tools",
		Usage: "Install ripgrep, fd, bat, yq, glow, and gh",
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			return tools.InstallAll(commandRunner(cmd, dryRun, demo, runnerFactory))
		}),
	}
}

func devToolsCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "dev",
		Usage: "Install Go, Node.js, Rust, and ecosystem tools",
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
			&cli.BoolFlag{
				Name:  "rust",
				Usage: "Install Rust via rustup for the target user",
			},
			&cli.BoolFlag{
				Name:  "go-lint",
				Usage: "Install golangci-lint",
			},
			&cli.BoolFlag{
				Name:  "goreleaser",
				Usage: "Install GoReleaser",
			},
			&cli.BoolFlag{
				Name:  "govulncheck",
				Usage: "Install govulncheck",
			},
			&cli.BoolFlag{
				Name:  "pnpm",
				Usage: "Install pnpm via Corepack for the target user",
			},
		},
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			username := cmd.String("user")
			opts := devtools.InstallOptions{
				Go:          cmd.Bool("go"),
				Node:        cmd.Bool("node"),
				Rust:        cmd.Bool("rust"),
				GoLint:      cmd.Bool("go-lint"),
				GoReleaser:  cmd.Bool("goreleaser"),
				GoVulnCheck: cmd.Bool("govulncheck"),
				Pnpm:        cmd.Bool("pnpm"),
			}
			if cmd.Bool("all") {
				opts = devtools.AllInstallOptions()
			} else if !opts.Any() {
				opts = devtools.DefaultInstallOptions()
			}
			return devtools.InstallSelected(commandRunner(cmd, dryRun, demo, runnerFactory), username, opts)
		}),
	}
}

func doctorCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Run read-only instance checks",
		Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
			report := diagnostics.Run(commandRunner(cmd, dryRun, demo, runnerFactory))
			fmt.Println(diagnostics.Format(report))
			return nil
		}),
	}
}

func firewallCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "network",
		Usage: "Manage UFW network rules",
		Commands: []*cli.Command{
			{
				Name:  "status",
				Usage: "Show UFW status",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := firewall.Status(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "list",
				Usage: "Show numbered UFW rules",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := firewall.StatusNumbered(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "enable",
				Usage: "Install and enable UFW with safe defaults",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "allow-ssh", Usage: "Allow the detected SSH port before enabling", Value: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return firewall.EnableBaseline(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.Bool("allow-ssh"))
				}),
			},
			{
				Name:  "allow",
				Usage: "Allow a TCP or UDP port",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "port", Usage: "Port or port range, e.g. 443 or 60000:61000", Required: true},
					&cli.StringFlag{Name: "proto", Usage: "Protocol: tcp or udp", Value: "tcp"},
					&cli.StringFlag{Name: "from", Usage: "Optional source IP or CIDR"},
					&cli.StringFlag{Name: "comment", Usage: "Optional UFW rule comment"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return firewall.AllowRule(commandRunner(cmd, dryRun, demo, runnerFactory), firewall.Rule{
						Port:    cmd.String("port"),
						Proto:   cmd.String("proto"),
						From:    cmd.String("from"),
						Comment: cmd.String("comment"),
					})
				}),
			},
			{
				Name:  "delete",
				Usage: "Delete a numbered UFW rule",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "number", Usage: "Rule number from network list", Required: true},
					&cli.BoolFlag{Name: "yes", Usage: "Confirm rule deletion"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("network delete requires --yes")
					}
					return firewall.DeleteRule(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.Int("number"))
				}),
			},
			{
				Name:  "reset",
				Usage: "Reset UFW rules",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Usage: "Confirm firewall reset"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("network reset requires --yes")
					}
					return firewall.Reset(commandRunner(cmd, dryRun, demo, runnerFactory))
				}),
			},
		},
	}
}

func fail2banCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "guard",
		Usage: "Manage fail2ban SSH protection",
		Commands: []*cli.Command{
			{
				Name:  "install",
				Usage: "Install fail2ban and configure the SSH jail",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "bantime", Value: "1h", Usage: "Ban duration"},
					&cli.StringFlag{Name: "findtime", Value: "10m", Usage: "Retry window"},
					&cli.IntFlag{Name: "maxretry", Value: 5, Usage: "Maximum retries before ban"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return security.InstallFail2Ban(commandRunner(cmd, dryRun, demo, runnerFactory), security.Fail2BanOptions{
						BanTime:  cmd.String("bantime"),
						FindTime: cmd.String("findtime"),
						MaxRetry: cmd.Int("maxretry"),
					})
				}),
			},
			{
				Name:  "status",
				Usage: "Show fail2ban SSH jail status",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := security.Fail2BanStatus(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "unban",
				Usage: "Unban an IP address from the SSH jail",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ip", Usage: "IP address to unban", Required: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return security.UnbanIP(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("ip"))
				}),
			},
		},
	}
}

func dockerCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "containers",
		Usage: "Manage Docker maintenance tasks",
		Commands: []*cli.Command{
			{
				Name:  "log-rotation",
				Usage: "Configure Docker json-file log rotation",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "max-size", Value: "10m", Usage: "Maximum log file size"},
					&cli.StringFlag{Name: "max-file", Value: "3", Usage: "Maximum rotated log files"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return dockermaint.ConfigureLogRotation(commandRunner(cmd, dryRun, demo, runnerFactory), dockermaint.LogRotationOptions{
						MaxSize: cmd.String("max-size"),
						MaxFile: cmd.String("max-file"),
					})
				}),
			},
			{
				Name:  "disk",
				Usage: "Show Docker disk usage",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := dockermaint.DiskUsage(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "prune",
				Usage: "Prune selected Docker resources",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "containers", Usage: "Prune stopped containers"},
					&cli.BoolFlag{Name: "images", Usage: "Prune dangling images"},
					&cli.BoolFlag{Name: "build-cache", Usage: "Prune build cache"},
					&cli.BoolFlag{Name: "yes", Usage: "Confirm Docker prune"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("containers prune requires --yes")
					}
					return dockermaint.Prune(commandRunner(cmd, dryRun, demo, runnerFactory), dockermaint.PruneOptions{
						Containers: cmd.Bool("containers"),
						Images:     cmd.Bool("images"),
						BuildCache: cmd.Bool("build-cache"),
					})
				}),
			},
		},
	}
}

func updatesCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "updates",
		Usage: "Manage package updates and reboot checks",
		Commands: []*cli.Command{
			{
				Name:  "check",
				Usage: "Update apt metadata and list upgradable packages",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := updates.Check(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "upgrade",
				Usage: "Run apt full-upgrade",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return updates.Upgrade(commandRunner(cmd, dryRun, demo, runnerFactory))
				}),
			},
			{
				Name:  "reboot-needed",
				Usage: "Show whether a reboot is required",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := updates.RebootRequired(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "unattended",
				Usage: "Show unattended-upgrades service status",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := updates.UnattendedStatus(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "failed-units",
				Usage: "Show failed systemd units",
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := updates.FailedUnits(commandRunner(cmd, dryRun, demo, runnerFactory))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "reboot",
				Usage: "Reboot the instance",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Usage: "Confirm reboot"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return updates.Reboot(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.Bool("yes"))
				}),
			},
		},
	}
}

func serviceCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "service",
		Usage: "Manage setup-created per-user systemd services",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create and start a managed per-user systemd service",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
					&cli.StringFlag{Name: "workdir", Usage: "Absolute working directory", Required: true},
					&cli.StringFlag{Name: "cmd", Usage: "Command to run", Required: true},
					&cli.StringFlag{Name: "env-file", Usage: "Optional absolute EnvironmentFile path"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return service.Create(commandRunner(cmd, dryRun, demo, runnerFactory), service.Config{
						User:    cmd.String("user"),
						Name:    cmd.String("name"),
						WorkDir: cmd.String("workdir"),
						Command: cmd.String("cmd"),
						EnvFile: cmd.String("env-file"),
					})
				}),
			},
			{
				Name:  "status",
				Usage: "Show a managed user service status",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := service.Status(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("name"))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "list",
				Usage: "List setup-managed user services",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					units, err := service.List(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"))
					if err != nil {
						return err
					}
					if len(units) == 0 {
						fmt.Println("No setup-managed services found.")
						return nil
					}
					fmt.Println(strings.Join(units, "\n"))
					return nil
				}),
			},
			{
				Name:  "logs",
				Usage: "Show recent managed user service logs",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					out, err := service.Logs(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("name"))
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
				}),
			},
			{
				Name:  "restart",
				Usage: "Restart a managed user service",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					return service.Restart(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("name"))
				}),
			},
			{
				Name:  "disable",
				Usage: "Disable and stop a managed user service",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
					&cli.BoolFlag{Name: "yes", Usage: "Confirm disabling and stopping the service"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("service disable requires --yes")
					}
					return service.Disable(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("name"))
				}),
			},
			{
				Name:  "remove",
				Usage: "Remove a managed user service unit",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "Target user", Required: true},
					&cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
					&cli.BoolFlag{Name: "yes", Usage: "Confirm disabling and removing the service unit"},
				},
				Action: provisioningAction(func(ctx context.Context, cmd *cli.Command) error {
					if !cmd.Bool("yes") {
						return fmt.Errorf("service remove requires --yes")
					}
					return service.Remove(commandRunner(cmd, dryRun, demo, runnerFactory), cmd.String("user"), cmd.String("name"))
				}),
			},
		},
	}
}

func fullCmd(dryRun, demo bool, runnerFactory RunnerFactory) *cli.Command {
	return &cli.Command{
		Name:  "fresh",
		Usage: "Run the full fresh-instance setup",
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
			runner := commandRunner(cmd, dryRun, demo, runnerFactory)

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
		Name:  "version",
		Usage: "Print version info",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cmd.Root().Version)
			return nil
		},
	}
}
