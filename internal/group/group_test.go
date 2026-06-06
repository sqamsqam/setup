package group

import (
	"errors"
	"os"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type groupTestRunner struct {
	setupexec.DryRunner
	outputs map[string]string
	runs    []string
}

func (r *groupTestRunner) Output(name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	out, ok := r.outputs[key]
	if !ok {
		return "", errors.New("not found")
	}
	return out, nil
}

func (r *groupTestRunner) Run(name string, args ...string) error {
	r.runs = append(r.runs, name+" "+strings.Join(args, " "))
	return nil
}

func (r *groupTestRunner) LookupUser(username string) (uid, gid int, err error) {
	if username == "dev" {
		return 1000, 1000, nil
	}
	return 0, 0, os.ErrNotExist
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"app", false},
		{"app_ops", false},
		{"app-ops", false},
		{"", true},
		{"App", true},
		{"1app", true},
		{strings.Repeat("a", 33), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestCreateSkipsExistingGroup(t *testing.T) {
	r := &groupTestRunner{outputs: map[string]string{
		"getent group app": "app:x:2000:",
	}}

	if err := Create(r, "app"); err != nil {
		t.Fatal(err)
	}
	if len(r.runs) != 0 {
		t.Fatalf("expected no command for existing group, got %v", r.runs)
	}
}

func TestDeleteRefusesPrimaryGroup(t *testing.T) {
	r := &groupTestRunner{outputs: map[string]string{
		"getent group app":   "app:x:2000:",
		"getent passwd":      "dev:x:1000:2000:Dev:/home/dev:/bin/bash\nother:x:1001:1001:Other:/home/other:/bin/bash",
		"id -nG dev":         "dev app",
		"getent group other": "other:x:2001:",
	}}

	err := Delete(r, "app")
	if err == nil || !strings.Contains(err.Error(), "primary group") {
		t.Fatalf("expected primary group refusal, got %v", err)
	}
	if len(r.runs) != 0 {
		t.Fatalf("expected no delete command, got %v", r.runs)
	}
}

func TestAddUserSkipsExistingMembership(t *testing.T) {
	r := &groupTestRunner{outputs: map[string]string{
		"getent group app": "app:x:2000:dev",
		"id -nG dev":       "dev app",
	}}

	if err := AddUser(r, "dev", "app"); err != nil {
		t.Fatal(err)
	}
	if len(r.runs) != 0 {
		t.Fatalf("expected no command for existing membership, got %v", r.runs)
	}
}

func TestRemoveUserRunsGpasswd(t *testing.T) {
	r := &groupTestRunner{outputs: map[string]string{
		"getent group app": "app:x:2000:dev",
		"id -nG dev":       "dev app",
	}}

	if err := RemoveUser(r, "dev", "app"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.runs, "\n"); !strings.Contains(got, "gpasswd -d dev app") {
		t.Fatalf("expected gpasswd removal, got %q", got)
	}
}
