package user

import (
	"fmt"
	"regexp"
	"strings"
)

var usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

var validKeyPrefixes = []string{
	"ssh-rsa",
	"ssh-ed25519",
	"ssh-dss",
	"ecdsa-sha2-nistp256",
	"ecdsa-sha2-nistp384",
	"ecdsa-sha2-nistp521",
	"sk-ssh-ed25519",
	"sk-ecdsa-sha2-nistp256",
}

func ValidateUsername(name string) error {
	if name == "" {
		return fmt.Errorf("username must not be empty")
	}
	if !usernameRe.MatchString(name) {
		return fmt.Errorf("invalid username %q: must start with a lowercase letter or underscore, followed by letters, digits, hyphens, or underscores", name)
	}
	if len(name) > 32 {
		return fmt.Errorf("username too long: %d characters (max 32)", len(name))
	}
	return nil
}

func ValidateSSHKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("SSH public key must not be empty")
	}
	for _, prefix := range validKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return nil
		}
	}
	return fmt.Errorf("SSH public key does not start with a recognised key type prefix")
}
