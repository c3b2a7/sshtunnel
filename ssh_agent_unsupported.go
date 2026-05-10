//go:build !unix || android || ios

package main

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"golang.org/x/crypto/ssh/agent"
)

// dialSSHAgent reports that ssh-agent authentication is unavailable on this platform.
func dialSSHAgent(_ context.Context, _ string) (agent.ExtendedAgent, io.Closer, error) {
	return nil, nil, fmt.Errorf("ssh-agent authentication is not supported on %s", runtime.GOOS)
}
