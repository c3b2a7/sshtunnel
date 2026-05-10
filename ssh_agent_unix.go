//go:build unix && !android && !ios

package main

import (
	"context"
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh/agent"
)

// dialSSHAgent connects to a Unix ssh-agent socket.
func dialSSHAgent(ctx context.Context, authSock string) (agent.ExtendedAgent, io.Closer, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", authSock)
	if err != nil {
		return nil, nil, fmt.Errorf("connect %s: %w", authSock, err)
	}
	return agent.NewClient(conn), conn, nil
}
