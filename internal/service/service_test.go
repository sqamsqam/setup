package service

import (
	"strings"
	"testing"
)

func TestUnitName(t *testing.T) {
	if got := UnitName("app"); got != "setup-app.service" {
		t.Fatalf("unexpected unit name: %s", got)
	}
	if got := UnitName("setup-api"); got != "setup-api.service" {
		t.Fatalf("unexpected prefixed unit name: %s", got)
	}
}

func TestUnitContent(t *testing.T) {
	content := UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
		EnvFile: "/home/dev/app/.env",
	})
	for _, want := range []string{
		"Managed by setup",
		"WorkingDirectory=\"/home/dev/app\"",
		"EnvironmentFile=-\"/home/dev/app/.env\"",
		"ExecStart=/bin/bash -lc \"npm start\"",
		"Restart=on-failure",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in %q", want, content)
		}
	}
}
