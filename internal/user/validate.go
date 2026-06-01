package user

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

var usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

var reservedUsernames = map[string]bool{
	"root": true, "daemon": true, "bin": true, "sys": true,
	"sync": true, "games": true, "man": true, "lp": true,
	"mail": true, "news": true, "uucp": true, "proxy": true,
	"www-data": true, "backup": true, "list": true, "irc": true,
	"gnats": true, "nobody": true, "sshd": true,
}

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
	lower := strings.ToLower(name)
	if reservedUsernames[lower] {
		return fmt.Errorf("username %q is reserved", name)
	}
	if strings.HasPrefix(lower, "systemd-") {
		return fmt.Errorf("username %q uses reserved prefix", name)
	}
	return nil
}

func ValidateSSHKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("SSH public key must not be empty")
	}
	var matched bool
	for _, prefix := range validKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("SSH public key does not start with a recognised key type prefix")
	}

	fields := strings.Fields(key)
	if len(fields) < 2 {
		return fmt.Errorf("SSH public key must have at least 2 fields (type and base64 data)")
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return fmt.Errorf("SSH public key data is not valid base64: %w", err)
	}
	if len(decoded) < 16 {
		return fmt.Errorf("SSH public key data is too short (%d bytes, minimum 16)", len(decoded))
	}
	return nil
}

func SSHKeySummary(key string) string {
	fields := strings.Fields(strings.TrimSpace(key))
	if len(fields) < 2 {
		return strings.TrimSpace(key)
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return fields[0]
	}
	sum := sha256.Sum256(decoded)
	fingerprint := base64.RawStdEncoding.EncodeToString(sum[:])
	if len(fields) > 2 {
		return fmt.Sprintf("%s SHA256:%s %s", fields[0], fingerprint, strings.Join(fields[2:], " "))
	}
	return fmt.Sprintf("%s SHA256:%s", fields[0], fingerprint)
}
