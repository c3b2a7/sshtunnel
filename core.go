package main

import (
	"errors"
	"fmt"
	"github.com/c3b2a7/sshtunnel/socks"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Endpoint struct {
	Host, Port string
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s:%s", e.Host, e.Port)
}

type Mode int

func (m Mode) String() string {
	switch m {
	case Local:
		return "LOCAL"
	case Remote:
		return "REMOTE"
	case Dynamic:
		return "DYNAMIC"
	}
	return "UNSUPPORTED"
}

const (
	Local Mode = 1 << iota
	Remote
	Dynamic
)

var ModeNotSupported = errors.New("mode not supported, available: local, remote, dynamic")

func PickMode(mode string) (Mode, error) {
	mode = strings.ToUpper(mode)
	switch mode {
	case "LOCAL":
		return Local, nil
	case "REMOTE":
		return Remote, nil
	case "DYNAMIC":
		return Dynamic, nil
	}
	return -1, ModeNotSupported
}

type Tunnel struct {
	Remote Endpoint
	Local  Endpoint
	Mode   Mode
}

type Config struct {
	Target     Endpoint
	Username   string
	PrivateKey []byte
	Passphrase string
}

func NewEndpoint(addr string) (Endpoint, error) {
	host, port, err := net.SplitHostPort(addr)
	return Endpoint{
		Host: host,
		Port: port,
	}, err
}

func Connect(config Config) (*ssh.Client, error) {
	var auth []ssh.AuthMethod

	if config.PrivateKey != nil && config.Passphrase != "" {
		privateKey, _ := ssh.ParsePrivateKeyWithPassphrase(config.PrivateKey, []byte(config.Passphrase))
		auth = append(auth, ssh.PublicKeys(privateKey))
	} else if config.PrivateKey != nil {
		privateKey, _ := ssh.ParsePrivateKey(config.PrivateKey)
		auth = append(auth, ssh.PublicKeys(privateKey))
	} else if config.Passphrase != "" {
		auth = append(auth, ssh.Password(config.Passphrase))
	}

	return ssh.Dial("tcp", config.Target.String(), &ssh.ClientConfig{
		User:            config.Username,
		Auth:            auth,
		Timeout:         10 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

// Bridge
func (t Tunnel) Bridge(conn *ssh.Client) error {
	l, err := listen(t, conn)
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			logger.Printf("failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			var rc net.Conn
			var local, remote string

			switch t.Mode {
			case Local:
				local, remote = t.Local.String(), t.Remote.String()
				rc, err = conn.Dial("tcp", remote)
			case Remote:
				local, remote = t.Remote.String(), t.Local.String()
				rc, err = net.Dial("tcp", remote)
			case Dynamic:
				tgt, err := socks.Handshake(c)
				if err != nil && err == socks.InfoUDPAssociate {
					buf := make([]byte, 1)
					for {
						// block here
						_, err := c.Read(buf)
						if err, ok := err.(net.Error); ok && err.Timeout() {
							continue
						}
						logger.Printf("UDP Associate End.")
						return
					}
				}
				local, remote = t.Local.String(), tgt.String()
				rc, err = conn.Dial("tcp", remote)
			default:
				err = ModeNotSupported
			}

			if err != nil {
				logger.Printf("failed to connect to remote %s via server %s: %v", remote, conn.RemoteAddr(), err)
				return
			}
			defer rc.Close()
			logger.Printf("tunneling(%s) %s <-> %s <-> %s", t.Mode.String(), local, conn.RemoteAddr(), remote)
			if err = relay(rc, c); err != nil {
				logger.Printf("relay error: %v", err)
			}
		}()
	}
}

func listen(t Tunnel, conn *ssh.Client) (net.Listener, error) {
	switch t.Mode {
	case Local, Dynamic:
		return net.Listen("tcp", t.Local.String())
	case Remote:
		return conn.Listen("tcp", t.Remote.String())
	}
	return nil, ModeNotSupported
}

// relay copies between left and right bidirectionally
func relay(left, right net.Conn) error {
	var err, err1 error
	var wg sync.WaitGroup
	var wait = 5 * time.Second
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err1 = io.Copy(right, left)
		right.SetReadDeadline(time.Now().Add(wait)) // unblock read on right
	}()
	_, err = io.Copy(left, right)
	left.SetReadDeadline(time.Now().Add(wait)) // unblock read on left
	wg.Wait()
	if err1 != nil && !errors.Is(err1, os.ErrDeadlineExceeded) { // requires Go 1.15+
		return err1
	}
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		return err
	}
	return nil
}
