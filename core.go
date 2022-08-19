package main

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var app struct {
	Verbose bool
	Config  Config
}

type Endpoint struct {
	Host, Port string
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s:%s", e.Host, e.Port)
}

type Tunnel struct {
	Remote Endpoint
	Local  Endpoint
}

type Config struct {
	Tunnels    []Tunnel
	Target     Endpoint
	Username   string
	PrivateKey []byte
	Passphrase string
}

func NewEndpoint(addr string) (Endpoint, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatalln(err)
	}
	return Endpoint{
		Host: host,
		Port: port,
	}, nil
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
	l, err := net.Listen("tcp", t.Local.String())
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			logf("failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			rc, err := conn.Dial("tcp", t.Remote.String())
			if err != nil {
				logf("failed to connect to remote %s via server %s: %v", t.Remote, conn.RemoteAddr(), err)
				return
			}
			defer rc.Close()

			logf("tunneling %s <-> %s <-> %s", c.LocalAddr(), conn.RemoteAddr(), t.Remote.String())
			if err = relay(rc, c); err != nil {
				logf("relay error: %v", err)
			}
		}()
	}
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
