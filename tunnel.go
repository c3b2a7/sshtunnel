package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/c3b2a7/sshtunnel/internal/log"
	"github.com/c3b2a7/sshtunnel/internal/socks"
	"golang.org/x/crypto/ssh"
)

var (
	ErrUnsupportedMode = errors.New("mode not supported, available: local, remote, dynamic")
	ErrSkipRelay       = errors.New("skip relay")
)

type NetworkAddress struct {
	Host string
	Port string
}

func ParseNetworkAddress(value string) (NetworkAddress, error) {
	value = strings.TrimSpace(value)
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return NetworkAddress{}, err
	}
	return NetworkAddress{Host: host, Port: port}, nil
}

func (a NetworkAddress) String() string {
	return net.JoinHostPort(a.Host, a.Port)
}

type ForwardMode string

const (
	ForwardLocal   ForwardMode = "local"
	ForwardRemote  ForwardMode = "remote"
	ForwardDynamic ForwardMode = "dynamic"
)

func ParseForwardMode(value string) (ForwardMode, error) {
	mode := ForwardMode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case ForwardLocal, ForwardRemote, ForwardDynamic:
		return mode, nil
	default:
		return "", ErrUnsupportedMode
	}
}

func (m ForwardMode) String() string {
	if m == "" {
		return "unsupported"
	}
	return string(m)
}

type TunnelSpec struct {
	Local  NetworkAddress
	Remote NetworkAddress
	Mode   ForwardMode
}

type SSHConfig struct {
	Server     NetworkAddress
	Username   string
	PrivateKey []byte
	Passphrase string
}

func DialSSH(config SSHConfig) (*ssh.Client, error) {
	auth, err := buildAuthMethods(config)
	if err != nil {
		return nil, err
	}

	return ssh.Dial("tcp", config.Server.String(), &ssh.ClientConfig{
		User:            config.Username,
		Auth:            auth,
		Timeout:         10 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

func buildAuthMethods(config SSHConfig) ([]ssh.AuthMethod, error) {
	switch {
	case len(config.PrivateKey) > 0 && config.Passphrase != "":
		signer, err := ssh.ParsePrivateKeyWithPassphrase(config.PrivateKey, []byte(config.Passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	case len(config.PrivateKey) > 0:
		signer, err := ssh.ParsePrivateKey(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	case config.Passphrase != "":
		return []ssh.AuthMethod{ssh.Password(config.Passphrase)}, nil
	default:
		return nil, nil
	}
}

// ServeTunnel accepts tunnel connections and relays each connection in its own goroutine.
func ServeTunnel(ctx context.Context, client *ssh.Client, tunnel TunnelSpec) error {
	listener, err := listenTunnel(client, tunnel)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
		}
		_ = listener.Close()
	}()

	go func() {
		defer close(done)
		for {
			clientConn, acceptErr := listener.Accept()
			if acceptErr != nil {
				if ctx.Err() != nil || errors.Is(acceptErr, net.ErrClosed) {
					return
				}
				log.Errorf(ctx, "failed to accept: %s", acceptErr)
				continue
			}

			select {
			case <-ctx.Done():
				_ = clientConn.Close()
				return
			default:
				go handleTunnelConn(ctx, client, tunnel, clientConn)
			}
		}
	}()

	return nil
}

func listenTunnel(client *ssh.Client, tunnel TunnelSpec) (net.Listener, error) {
	switch tunnel.Mode {
	case ForwardLocal, ForwardDynamic:
		return net.Listen("tcp", tunnel.Local.String())
	case ForwardRemote:
		return client.Listen("tcp", tunnel.Remote.String())
	default:
		return nil, ErrUnsupportedMode
	}
}

func handleTunnelConn(ctx context.Context, client *ssh.Client, tunnel TunnelSpec, clientConn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf(ctx, "panic: %v", err)
		}
	}()
	defer clientConn.Close()

	targetConn, targetAddr, err := openTunnelTarget(ctx, client, tunnel, clientConn)
	if err != nil {
		if errors.Is(err, ErrSkipRelay) {
			return
		}
		log.Errorf(ctx, "failed to connect to %s via server %s: %v", targetAddr, client.RemoteAddr(), err)
		return
	}
	if targetConn == nil {
		log.Errorf(ctx, "failed to connect to %s via server %s: empty connection", targetAddr, client.RemoteAddr())
		return
	}
	defer targetConn.Close()

	log.Debugf(
		ctx, "tunneling(%s) %s -> %s <-> %s -> %s",
		tunnel.Mode,
		client.LocalAddr(), client.RemoteAddr(),
		targetConn.LocalAddr(), targetConn.RemoteAddr(),
	)
	if err = relay(ctx, targetConn, clientConn); err != nil {
		log.Errorf(ctx, "relay error: %v", err)
	}
}

func openTunnelTarget(ctx context.Context, client *ssh.Client, tunnel TunnelSpec, clientConn net.Conn) (net.Conn, string, error) {
	switch tunnel.Mode {
	case ForwardLocal:
		return openLocalTarget(ctx, client, tunnel)
	case ForwardRemote:
		return openRemoteTarget(ctx, tunnel)
	case ForwardDynamic:
		return openDynamicTarget(ctx, client, tunnel, clientConn)
	default:
		return nil, "", ErrUnsupportedMode
	}
}

func openLocalTarget(ctx context.Context, client *ssh.Client, tunnel TunnelSpec) (net.Conn, string, error) {
	targetAddr := tunnel.Remote.String()
	conn, err := client.DialContext(ctx, "tcp", targetAddr)
	return conn, targetAddr, err
}

func openRemoteTarget(ctx context.Context, tunnel TunnelSpec) (net.Conn, string, error) {
	targetAddr := tunnel.Local.String()
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	return conn, targetAddr, err
}

func openDynamicTarget(ctx context.Context, client *ssh.Client, tunnel TunnelSpec, clientConn net.Conn) (net.Conn, string, error) {
	targetAddr, err := socks.Handshake(clientConn)
	if errors.Is(err, socks.InfoUDPAssociate) {
		waitUDPAssociate(ctx, clientConn)
		return nil, "", ErrSkipRelay
	}
	if err != nil {
		return nil, "", err
	}

	conn, err := client.DialContext(ctx, "tcp", targetAddr.String())
	return conn, targetAddr.String(), err
}

func waitUDPAssociate(ctx context.Context, conn net.Conn) {
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	buf := make([]byte, 1)
	for {
		if _, err := conn.Read(buf); err != nil {
			if ctx.Err() != nil {
				return
			}
			if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
				continue
			}
			log.Infof(ctx, "UDP Associate End.")
			return
		}
	}
}

var relayBufferPool = sync.Pool{
	New: func() any {
		return new(make([]byte, 32*1024))
	},
}

// relay copies data in both directions until either side closes or timeout.
func relay(ctx context.Context, left, right net.Conn) error {
	var firstErr, secondErr error
	var wg sync.WaitGroup
	wait := 5 * time.Second
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			_ = left.Close()
			_ = right.Close()
		case <-done:
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		secondErr = copyBuffer(right, left)
		_ = right.SetReadDeadline(time.Now().Add(wait))
	}()

	firstErr = copyBuffer(left, right)
	_ = left.SetReadDeadline(time.Now().Add(wait))
	wg.Wait()

	if ctx.Err() != nil {
		return nil
	}
	if secondErr != nil && !errors.Is(secondErr, os.ErrDeadlineExceeded) {
		return secondErr
	}
	if firstErr != nil && !errors.Is(firstErr, os.ErrDeadlineExceeded) {
		return firstErr
	}
	return nil
}

func copyBuffer(dst, src net.Conn) error {
	buf := relayBufferPool.Get().(*[]byte)
	defer relayBufferPool.Put(buf)
	_, err := io.CopyBuffer(dst, src, *buf)
	return err
}
