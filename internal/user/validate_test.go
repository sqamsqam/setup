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
