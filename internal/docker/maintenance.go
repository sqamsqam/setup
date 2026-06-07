package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
)

const DaemonConfigPath = "/etc/docker/daemon.json"

type LogRotationOptions struct {
	MaxSize string
	MaxFile string
}

type PruneOptions struct {
	Containers bool
	Images     bool
	BuildCache bool
}

func DefaultLogRotationOptions() LogRotationOptions {
	return LogRotationOptions{MaxSize: "10m", MaxFile: "3"}
}

func ConfigureLogRotation(runner setupexec.CmdRunner, opts LogRotationOptions) error {
	changed, err := WriteLogRotationConfig(runner, DaemonConfigPath, opts)
	if err != nil {
		return err
	}
	if changed {
		return runner.Run("systemctl", "restart", "docker")
	}
	return nil
}

func WriteLogRotationConfig(runner setupexec.CmdRunner, path string, opts LogRotationOptions) (bool, error) {
	data, err := MergeLogRotationConfig(runner, path, opts)
	if err != nil {
		return false, err
	}
	if err := validateDaemonConfig(runner, path, data); err != nil {
		return false, err
	}
	return managed.WriteFileIfChanged(runner, path, data, 0644)
}

func MergeLogRotationConfig(runner setupexec.CmdRunner, path string, opts LogRotationOptions) ([]byte, error) {
	if opts.MaxSize == "" {
		opts.MaxSize = "10m"
	}
	if opts.MaxFile == "" {
		opts.MaxFile = "3"
	}
	if err := validateLogRotationOption("max-size", opts.MaxSize); err != nil {
		return nil, err
	}
	if err := validateLogRotationOption("max-file", opts.MaxFile); err != nil {
		return nil, err
	}

	config := map[string]any{}
	oldContent, err := runner.ReadFile(path)
	if err == nil && len(oldContent) > 0 {
		if err := json.Unmarshal(oldContent, &config); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	config["log-driver"] = "json-file"
	logOpts, _ := config["log-opts"].(map[string]any)
	if logOpts == nil {
		logOpts = map[string]any{}
	}
	logOpts["max-size"] = opts.MaxSize
	logOpts["max-file"] = opts.MaxFile
	config["log-opts"] = logOpts

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func validateLogRotationOption(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s must be a single line", name)
	}
	return nil
}

func validateDaemonConfig(runner setupexec.CmdRunner, path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := runner.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpPath, err := runner.CreateTemp(dir, ".setup-docker-daemon-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = runner.Remove(tmpPath) }()
	if err := runner.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	if err := runner.Chmod(tmpPath, 0644); err != nil {
		return err
	}
	if err := runner.Run("dockerd", "--validate", "--config-file", tmpPath); err != nil {
		return fmt.Errorf("validate Docker daemon config: %w", err)
	}
	return nil
}

func DiskUsage(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("docker", "system", "df")
}

func Prune(runner setupexec.CmdRunner, opts PruneOptions) error {
	if !opts.Containers && !opts.Images && !opts.BuildCache {
		return fmt.Errorf("select at least one prune target")
	}
	if opts.Containers {
		if err := runner.Run("docker", "container", "prune", "-f"); err != nil {
			return err
		}
	}
	if opts.Images {
		if err := runner.Run("docker", "image", "prune", "-f"); err != nil {
			return err
		}
	}
	if opts.BuildCache {
		if err := runner.Run("docker", "builder", "prune", "-f"); err != nil {
			return err
		}
	}
	return nil
}
