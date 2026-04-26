package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	body := `{
		"target": "ssh.example.com:22",
		"username": "alice",
		"passphrase": "secret",
		"tunnels": [
			{"local": "127.0.0.1:13306", "remote": "10.0.0.2:3306", "mode": "local"},
			{"local": "127.0.0.1:1080", "mode": "dynamic"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.SSH.Server.String() != "ssh.example.com:22" {
		t.Fatalf("SSH server = %q, want %q", config.SSH.Server.String(), "ssh.example.com:22")
	}
	if config.SSH.Username != "alice" {
		t.Fatalf("SSH username = %q, want %q", config.SSH.Username, "alice")
	}
	if len(config.Tunnels) != 2 {
		t.Fatalf("tunnel count = %d, want 2", len(config.Tunnels))
	}
	if config.Tunnels[0].Mode != ForwardLocal {
		t.Fatalf("first tunnel mode = %q, want %q", config.Tunnels[0].Mode, ForwardLocal)
	}
	if config.Tunnels[1].Mode != ForwardDynamic {
		t.Fatalf("second tunnel mode = %q, want %q", config.Tunnels[1].Mode, ForwardDynamic)
	}
}

func TestLoadConfigRejectsInvalidTunnel(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	body := `{
		"target": "ssh.example.com:22",
		"username": "alice",
		"tunnels": [
			{"local": "127.0.0.1:1080", "mode": "invalid"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(configPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigRejectsLocalTunnelWithoutRemote(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	body := `{
		"target": "ssh.example.com:22",
		"username": "alice",
		"tunnels": [
			{"local": "127.0.0.1:13306", "mode": "local"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(configPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigRejectsEmptyTunnelList(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	body := `{
		"target": "ssh.example.com:22",
		"username": "alice"
	}`

	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(configPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}
