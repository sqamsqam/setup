package docker

import (
	"encoding/json"
	"os"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type fileRunner struct {
	*setupexec.DryRunner
	files map[string][]byte
}

func (f *fileRunner) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
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
