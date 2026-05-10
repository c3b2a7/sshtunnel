package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"path/filepath"
	"testing"
)

func TestBuildAuthMethodSpecsPrivateKeyOnly(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	_, privateKeyPKCS8 := marshalTestPrivateKey(t)
	auth := buildTestAuthMethods(t, SSHConfig{PrivateKey: privateKeyPKCS8})
	if len(auth.specs) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodPrivateKey {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodPrivateKey)
	}
}

func TestBuildAuthMethodSpecsPasswordOnly(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	auth := buildTestAuthMethods(t, SSHConfig{Passphrase: "secret"})
	if len(auth.specs) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodPassword {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodPassword)
	}
}

func TestBuildAuthMethodSpecsSkipsStaleEnvAgent(t *testing.T) {
	staleSocketPath := filepath.Join(t.TempDir(), "agent.sock")
	t.Setenv("SSH_AUTH_SOCK", staleSocketPath)

	_, privateKeyPKCS8 := marshalTestPrivateKey(t)
	auth := buildTestAuthMethods(t, SSHConfig{PrivateKey: privateKeyPKCS8})
	if len(auth.specs) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodPrivateKey {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodPrivateKey)
	}

	auth = buildTestAuthMethods(t, SSHConfig{Passphrase: "secret"})
	if len(auth.specs) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodPassword {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodPassword)
	}

	if _, err := buildAuthMethods(t.Context(), SSHConfig{
		SSHAuthSock: staleSocketPath,
		Passphrase:  "secret",
	}); err == nil {
		t.Fatal("buildAuthMethods() with explicit stale agent error = nil, want error")
	}
}

func TestBuildAuthMethodSpecsReturnsCanceledContext(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "agent.sock"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := buildAuthMethods(ctx, SSHConfig{Passphrase: "secret"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("buildAuthMethods() error = %v, want %v", err, context.Canceled)
	}
}

func TestDialSSHReturnsCanceledContext(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := DialSSH(ctx, SSHConfig{
		Server:     NetworkAddress{Host: "127.0.0.1", Port: "1"},
		Username:   "alice",
		Passphrase: "secret",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DialSSH() error = %v, want %v", err, context.Canceled)
	}
}

func TestResolveSSHAuthSock(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "/tmp/from-env.sock")

	if got, implicit := resolveSSHAuthSock(SSHConfig{SSHAuthSock: "/tmp/from-config.sock"}); got != "/tmp/from-config.sock" || implicit {
		t.Fatalf("resolveSSHAuthSock() = %q, want %q", got, "/tmp/from-config.sock")
	}
	if got, implicit := resolveSSHAuthSock(SSHConfig{}); got != "/tmp/from-env.sock" || !implicit {
		t.Fatalf("resolveSSHAuthSock() = %q, want %q", got, "/tmp/from-env.sock")
	}
}

func buildTestAuthMethods(t *testing.T, config SSHConfig) *authMethods {
	t.Helper()

	auth, err := buildAuthMethods(t.Context(), config)
	if err != nil {
		t.Fatalf("buildAuthMethods() error = %v", err)
	}
	t.Cleanup(func() {
		if err := auth.close(); err != nil {
			t.Fatalf("auth.close() error = %v", err)
		}
	})
	return auth
}

func marshalTestPrivateKey(t *testing.T) (ed25519.PrivateKey, []byte) {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	return privateKey, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
}
