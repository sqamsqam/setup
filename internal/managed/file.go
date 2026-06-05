package managed

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

const Marker = "# Managed by setup — do not edit\n"

func IsMarked(data []byte) bool {
	return bytes.HasPrefix(data, []byte(Marker))
}

func WriteManagedFileIfChanged(runner setupexec.CmdRunner, path string, data []byte, perm os.FileMode) (bool, error) {
	if !IsMarked(data) {
		return false, fmt.Errorf("managed content for %s must start with setup marker", path)
	}

	oldContent, err := runner.ReadFile(path)
	if err == nil {
		if bytes.Equal(oldContent, data) {
			return false, nil
		}
		if !IsMarked(oldContent) && (!setupexec.IsDryRun(runner) || len(oldContent) != 0) {
			return false, fmt.Errorf("refusing to replace unmanaged file %s", path)
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return writeFile(runner, path, data, perm)
}

func WriteFileIfChanged(runner setupexec.CmdRunner, path string, data []byte, perm os.FileMode) (bool, error) {
	oldContent, err := runner.ReadFile(path)
	if err == nil {
		if bytes.Equal(oldContent, data) {
			return false, nil
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return writeFile(runner, path, data, perm)
}

func writeFile(runner setupexec.CmdRunner, path string, data []byte, perm os.FileMode) (bool, error) {
	dir := filepath.Dir(path)
	if err := runner.MkdirAll(dir, 0755); err != nil {
		return false, err
	}

	tmpPath, err := runner.CreateTemp(dir, ".setup-*")
	if err != nil {
		return false, err
	}
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, data, perm); err != nil {
		return false, err
	}
	if err := runner.Chmod(tmpPath, perm); err != nil {
		return false, err
	}
	if err := runner.Rename(tmpPath, path); err != nil {
		return false, err
	}
	return true, nil
}
