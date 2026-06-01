package user

import (
	"errors"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"myuser", false},
		{"my_user", false},
		{"myuser123", false},
		{"my-user", false},
		{"_underscore", false},
		{"a", false},
		{"a-b_c1", false},
		{"", true},
		{"-dashfirst", true},
		{"123start", true},
		{"UPPER", true},
		{" spaces ", true},
		{"with$pecial", true},
		{string(make([]byte, 33)), true},
		{"root", true},
		{"ROOT", true},
		{"Root", true},
		{"nobody", true},
		{"daemon", true},
		{"sshd", true},
		{"systemd-network", true},
		{"systemd-resolve", true},
		{"systemd-timesyncd", true},
		{"systemd-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSSHKey(t *testing.T) {
	valid32 := "/B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo="
	valid57 := "DZZSN/oGYqFdRIPyP5gWaN/bHgqnD34e9xHclr1ZpSP5T56zYTHzjMkjh8wybwp1lGGtAYb5qOlN"

	tests := []struct {
		key     string
		wantErr bool
	}{
		{"ssh-ed25519 " + valid32, false},
		{"ssh-rsa " + valid57, false},
		{"ssh-dss " + valid32, false},
		{"ecdsa-sha2-nistp256 " + valid32, false},
		{"ecdsa-sha2-nistp384 " + valid32, false},
		{"ecdsa-sha2-nistp521 " + valid32, false},
		{"sk-ssh-ed25519 " + valid32, false},
		{"sk-ecdsa-sha2-nistp256 " + valid32, false},
		{"", true},
		{"invalid key", true},
		{"not-a-key-prefix " + valid57, true},
		{"ssh-ed25519", true},
		{"ssh-ed25519 invalid!!!base64", true},
	}

	for _, tt := range tests {
		t.Run(tt.key[:min(len(tt.key), 30)], func(t *testing.T) {
			err := ValidateSSHKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSSHKey(%q) error = %v, wantErr = %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

type passwdRunner struct {
	*setupexec.DryRunner
	passwd string
	err    error
}

func (p passwdRunner) Output(name string, args ...string) (string, error) {
	if name == "getent" {
		return p.passwd, p.err
	}
	return p.DryRunner.Output(name, args...)
}

func TestLookupAccountRejectsSystemUID(t *testing.T) {
	runner := passwdRunner{
		DryRunner: setupexec.NewDryRunner(),
		passwd:    "daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin",
	}
	_, err := lookupAccount(runner, "daemon")
	if err == nil {
		t.Fatal("expected low UID error")
	}
	if !strings.Contains(err.Error(), "below 1000") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLookupAccountUsesPasswdHome(t *testing.T) {
	runner := passwdRunner{
		DryRunner: setupexec.NewDryRunner(),
		passwd:    "dev:x:1001:1002:Dev:/srv/dev:/bin/bash",
	}
	acct, err := lookupAccount(runner, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if acct.home != "/srv/dev" {
		t.Fatalf("expected passwd home, got %q", acct.home)
	}
	if acct.uid != 1001 || acct.gid != 1002 {
		t.Fatalf("unexpected ids: uid=%d gid=%d", acct.uid, acct.gid)
	}
}

func TestLookupAccountPropagatesMissingUser(t *testing.T) {
	runner := passwdRunner{
		DryRunner: setupexec.NewDryRunner(),
		err:       errors.New("missing"),
	}
	_, err := lookupAccount(runner, "missing")
	if err == nil {
		t.Fatal("expected missing user error")
	}
}
