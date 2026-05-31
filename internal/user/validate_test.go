package user

import (
	"testing"
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
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...", false},
		{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...", false},
		{"ssh-dss AAAAB3NzaC1kc3MAAACBA...", false},
		{"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...", false},
		{"ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQ...", false},
		{"ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjE...", false},
		{"sk-ssh-ed25519 AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29t...", false},
		{"sk-ecdsa-sha2-nistp256 AAAAGnNrLWVjZHNhLXNoYTItbmlzdHAyNTZAb3BlbnNza...", false},
		{"", true},
		{"invalid key", true},
		{"not-a-key-prefix AAAAB3...", true},
		{"ssh-ed25519", false},
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
