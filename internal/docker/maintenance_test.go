package docker

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type fileRunner struct {
	*setupexec.DryRunner
	files   map[string][]byte
	readErr error
	ops     []string
}

func (f *fileRunner) ReadFile(path string) ([]byte, error) {
	f.ops = append(f.ops, "read:"+path)
	if f.readErr != nil {
		return nil, f.readErr
	}
	data, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (f *fileRunner) Run(name string, args ...string) error {
	f.ops = append(f.ops, "run:"+name+" "+strings.Join(args, " "))
	return nil
}

func TestMergeLogRotationConfigPreservesExistingKeys(t *testing.T) {
	runner := &fileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files: map[string][]byte{
			DaemonConfigPath: []byte(`{"storage-driver":"overlay2","log-opts":{"labels":"app"}}`),
		},
	}

	data, err := MergeLogRotationConfig(runner, DaemonConfigPath, DefaultLogRotationOptions())
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["storage-driver"] != "overlay2" {
		t.Fatalf("expected storage-driver preserved, got %#v", got)
	}
	opts := got["log-opts"].(map[string]any)
	if opts["labels"] != "app" || opts["max-size"] != "10m" || opts["max-file"] != "3" {
		t.Fatalf("unexpected log opts: %#v", opts)
	}
}

func TestPruneRequiresTarget(t *testing.T) {
	err := Prune(setupexec.NewDryRunner(), PruneOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigureLogRotationReturnsReadErrorWithoutRestart(t *testing.T) {
	runner := &fileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     map[string][]byte{},
		readErr:   errors.New("permission denied"),
	}

	err := ConfigureLogRotation(runner, DefaultLogRotationOptions())
	if err == nil {
		t.Fatal("expected read error")
	}
	if hasDockerRestart(runner.ops) {
		t.Fatalf("unexpected Docker restart after read error: %v", runner.ops)
	}
}

func TestConfigureLogRotationRejectsInvalidJSONWithoutRestart(t *testing.T) {
	runner := &fileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files: map[string][]byte{
			DaemonConfigPath: []byte("{"),
		},
	}

	err := ConfigureLogRotation(runner, DefaultLogRotationOptions())
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if hasDockerRestart(runner.ops) {
		t.Fatalf("unexpected Docker restart after invalid JSON: %v", runner.ops)
	}
}

func TestConfigureLogRotationSkipsRestartWhenUnchanged(t *testing.T) {
	data, err := json.MarshalIndent(map[string]any{
		"log-driver": "json-file",
		"log-opts": map[string]any{
			"max-size": "10m",
			"max-file": "3",
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files: map[string][]byte{
			DaemonConfigPath: append(data, '\n'),
		},
	}

	if err := ConfigureLogRotation(runner, DefaultLogRotationOptions()); err != nil {
		t.Fatal(err)
	}
	if hasDockerRestart(runner.ops) {
		t.Fatalf("unexpected Docker restart for unchanged config: %v", runner.ops)
	}
}

func TestConfigureLogRotationRestartsWhenChanged(t *testing.T) {
	runner := &fileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files: map[string][]byte{
			DaemonConfigPath: []byte(`{"storage-driver":"overlay2"}`),
		},
	}

	if err := ConfigureLogRotation(runner, DefaultLogRotationOptions()); err != nil {
		t.Fatal(err)
	}
	if !hasDockerRestart(runner.ops) {
		t.Fatalf("expected Docker restart after changed config: %v", runner.ops)
	}
}

func hasDockerRestart(ops []string) bool {
	for _, op := range ops {
		if op == "run:systemctl restart docker" {
			return true
		}
	}
	return false
}
