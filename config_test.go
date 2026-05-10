package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeConfigFile(t, cfgPath, fileConfig{
		Target:     "ssh.example.com:22",
		Username:   "alice",
		Passphrase: "secret",
		Tunnels: []tunnelConfig{
			{Local: "127.0.0.1:13306", Remote: "10.0.0.2:3306", Mode: "local"},
			{Local: "127.0.0.1:1080", Mode: "dynamic"},
		},
	})

	config, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.SSH.Server.String() != "ssh.example.com:22" {
		t.Fatalf("SSH server = %q, want %q", config.SSH.Server.String(), "ssh.example.com:22")
	}
	if config.SSH.Username != "alice" {
		t.Fatalf("SSH username = %q, want %q", config.SSH.Username, "alice")
	}
	if config.SSH.Passphrase != "secret" {
		t.Fatalf("SSH passphrase = %q, want %q", config.SSH.Passphrase, "secret")
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
	cfgPath := filepath.Join(dir, "config.json")
	writeConfigFile(t, cfgPath, fileConfig{
		Target:   "ssh.example.com:22",
		Username: "alice",
		Tunnels: []tunnelConfig{
			{Local: "127.0.0.1:1080", Mode: "invalid"},
		},
	})

	if _, err := LoadConfig(cfgPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigRejectsLocalTunnelWithoutRemote(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeConfigFile(t, cfgPath, fileConfig{
		Target:   "ssh.example.com:22",
		Username: "alice",
		Tunnels: []tunnelConfig{
			{Local: "127.0.0.1:13306", Mode: "local"},
		},
	})

	if _, err := LoadConfig(cfgPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigRejectsEmptyTunnelList(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeConfigFile(t, cfgPath, fileConfig{
		Target:   "ssh.example.com:22",
		Username: "alice",
	})

	if _, err := LoadConfig(cfgPath); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigSSHAuthSock(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	keyPath := filepath.Join(dir, "id_ed25519")
	agentSockPath := filepath.Join(dir, "agent.sock")

	key := []byte("test-key")
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, cfgPath, fileConfig{
		Target:      "ssh.example.com:22",
		Username:    "alice",
		PrivateKey:  keyPath,
		SSHAuthSock: agentSockPath,
		Tunnels: []tunnelConfig{
			{Local: "127.0.0.1:1080", Mode: "dynamic"},
		},
	})

	config, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.SSH.SSHAuthSock != agentSockPath {
		t.Fatalf("SSH auth sock = %q, want %q", config.SSH.SSHAuthSock, agentSockPath)
	}
	if !bytes.Equal(config.SSH.PrivateKey, key) {
		t.Fatalf("SSH private key = %q, want %q", string(config.SSH.PrivateKey), string(key))
	}
}

func writeConfigFile(t *testing.T, path string, config fileConfig) {
	t.Helper()

	body, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}
