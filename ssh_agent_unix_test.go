//go:build unix && !android && !ios

package main

import (
	"crypto/ed25519"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func TestBuildAuthMethodSpecsSSHAgentOnly(t *testing.T) {
	socketPath, _, stopAgent := startTestSSHAgent(t)
	t.Cleanup(stopAgent)
	t.Setenv("SSH_AUTH_SOCK", socketPath)

	auth := buildTestAuthMethods(t, SSHConfig{})
	if len(auth.specs) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodSSHAgent {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodSSHAgent)
	}
}

func TestBuildAuthMethodSpecsPrivateKeyThenAgent(t *testing.T) {
	socketPath, _, stopAgent := startTestSSHAgent(t)
	t.Cleanup(stopAgent)
	t.Setenv("SSH_AUTH_SOCK", socketPath)

	_, privateKeyPKCS8 := marshalTestPrivateKey(t)
	auth := buildTestAuthMethods(t, SSHConfig{
		PrivateKey: privateKeyPKCS8,
	})

	if len(auth.specs) != 2 {
		t.Fatalf("auth method count = %d, want 2", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodPrivateKey {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodPrivateKey)
	}
	if auth.specs[1].kind != authMethodSSHAgent {
		t.Fatalf("specs[1].kind = %q, want %q", auth.specs[1].kind, authMethodSSHAgent)
	}
}

func TestBuildAuthMethodSpecsAgentThenPassword(t *testing.T) {
	socketPath, _, stopAgent := startTestSSHAgent(t)
	t.Cleanup(stopAgent)

	t.Setenv("SSH_AUTH_SOCK", socketPath)

	auth := buildTestAuthMethods(t, SSHConfig{
		Passphrase: "secret",
	})

	if len(auth.specs) != 2 {
		t.Fatalf("auth method count = %d, want 2", len(auth.specs))
	}
	if auth.specs[0].kind != authMethodSSHAgent {
		t.Fatalf("specs[0].kind = %q, want %q", auth.specs[0].kind, authMethodSSHAgent)
	}
	if auth.specs[1].kind != authMethodPassword {
		t.Fatalf("specs[1].kind = %q, want %q", auth.specs[1].kind, authMethodPassword)
	}
}

func TestDialSSHAgent(t *testing.T) {
	socketPath, privateKey, stopAgent := startTestSSHAgent(t)
	defer stopAgent()

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	client, closer, err := dialSSHAgent(t.Context(), socketPath)
	if err != nil {
		t.Fatalf("dialSSHAgent() error = %v", err)
	}
	defer closer.Close()

	signers, err := client.Signers()
	if err != nil {
		t.Fatalf("agent Signers() error = %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("agent signer count = %d, want 1", len(signers))
	}
	if string(signers[0].PublicKey().Marshal()) != string(signer.PublicKey().Marshal()) {
		t.Fatal("agent signer public key mismatch")
	}
}

func startTestSSHAgent(t *testing.T) (string, ed25519.PrivateKey, func()) {
	t.Helper()

	socketDir, err := os.MkdirTemp("", "sshtunnel-*")
	if err != nil {
		t.Fatalf("create agent socket dir: %v", err)
	}

	socketPath := filepath.Join(socketDir, "agent.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = os.RemoveAll(socketDir)
		t.Fatalf("listen unix socket: %v", err)
	}

	keyring := agent.NewKeyring()
	privateKey, _ := marshalTestPrivateKey(t)
	if err = keyring.Add(agent.AddedKey{PrivateKey: privateKey}); err != nil {
		t.Fatalf("add key to keyring: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = agent.ServeAgent(keyring, conn)
	}()

	return socketPath, privateKey, func() {
		_ = listener.Close()
		<-done
		_ = os.RemoveAll(socketDir)
	}
}
