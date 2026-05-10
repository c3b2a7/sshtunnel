package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Server      NetworkAddress
	Username    string
	PrivateKey  []byte
	Passphrase  string
	SSHAuthSock string
}

type authMethodKind string

const (
	authMethodPrivateKey authMethodKind = "private-key"
	authMethodSSHAgent   authMethodKind = "ssh-agent"
	authMethodPassword   authMethodKind = "password"
)

type authMethodSpec struct {
	kind   authMethodKind
	method ssh.AuthMethod
	closer io.Closer
}

type authMethods struct {
	specs []authMethodSpec
}

func (auth *authMethods) methods() []ssh.AuthMethod {
	methods := make([]ssh.AuthMethod, 0, len(auth.specs))
	for _, spec := range auth.specs {
		methods = append(methods, spec.method)
	}
	return methods
}

func (auth *authMethods) close() error {
	var firstErr error
	for _, spec := range auth.specs {
		if spec.closer == nil {
			continue
		}
		if err := spec.closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// DialSSH opens an SSH client connection using the configured authentication methods.
func DialSSH(ctx context.Context, config SSHConfig) (*ssh.Client, error) {
	auth, err := buildAuthMethods(ctx, config)
	if err != nil {
		return nil, err
	}
	defer auth.close()

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            auth.methods(),
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", config.Server.String())
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, config.Server.String(), sshConfig)
	close(done)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err == nil {
			_ = clientConn.Close()
		} else {
			_ = conn.Close()
		}
		return nil, ctxErr
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}

// buildAuthMethods converts SSHConfig into ordered auth methods.
func buildAuthMethods(ctx context.Context, config SSHConfig) (*authMethods, error) {
	specs := make([]authMethodSpec, 0, 3)

	if len(config.PrivateKey) > 0 {
		method, err := buildPrivateKeyAuthMethod(config)
		if err != nil {
			return nil, err
		}
		specs = append(specs, authMethodSpec{
			kind:   authMethodPrivateKey,
			method: method,
		})
	}

	if authSock, implicit := resolveSSHAuthSock(config); authSock != "" {
		dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		agentClient, closer, err := dialSSHAgent(dialCtx, authSock)
		cancel()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if !implicit {
				return nil, err
			}
		} else {
			specs = append(specs, authMethodSpec{
				kind:   authMethodSSHAgent,
				method: ssh.PublicKeysCallback(agentClient.Signers),
				closer: closer,
			})
		}
	}

	if len(config.PrivateKey) == 0 && config.Passphrase != "" {
		specs = append(specs, authMethodSpec{
			kind:   authMethodPassword,
			method: ssh.Password(config.Passphrase),
		})
	}

	if len(specs) == 0 {
		return nil, errors.New("no ssh authentication method configured")
	}

	return &authMethods{specs}, nil
}

// buildPrivateKeyAuthMethod parses the configured private key and wraps it as an SSH auth method.
func buildPrivateKeyAuthMethod(config SSHConfig) (ssh.AuthMethod, error) {
	if config.Passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(config.PrivateKey, []byte(config.Passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w", err)
		}
		return ssh.PublicKeys(signer), nil
	}

	signer, err := ssh.ParsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

// resolveSSHAuthSock returns the configured agent socket or falls back to SSH_AUTH_SOCK.
func resolveSSHAuthSock(config SSHConfig) (string, bool) {
	if config.SSHAuthSock != "" {
		return config.SSHAuthSock, false
	}
	return os.Getenv("SSH_AUTH_SOCK"), true
}
