package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type AppConfig struct {
	SSH     SSHConfig
	Tunnels []TunnelSpec
}

var ErrNoTunnels = errors.New("at least one tunnel is required")

type fileConfig struct {
	Target      string         `json:"target"`
	Username    string         `json:"username"`
	PrivateKey  string         `json:"private-key"`
	Passphrase  string         `json:"passphrase"`
	SSHAuthSock string         `json:"ssh_auth_sock"`
	Tunnels     []tunnelConfig `json:"tunnels,omitempty"`
}

type tunnelConfig struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
	Mode   string `json:"mode"`
}

func LoadConfig(path string) (AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}

	var raw fileConfig
	if err = json.Unmarshal(data, &raw); err != nil {
		return AppConfig{}, err
	}

	return raw.toAppConfig()
}

func (c fileConfig) toAppConfig() (AppConfig, error) {
	server, err := ParseNetworkAddress(c.Target)
	if err != nil {
		return AppConfig{}, fmt.Errorf("parse target: %w", err)
	}

	config := AppConfig{
		SSH: SSHConfig{
			Server:      server,
			Username:    c.Username,
			Passphrase:  c.Passphrase,
			SSHAuthSock: c.SSHAuthSock,
		},
	}

	if c.PrivateKey != "" {
		config.SSH.PrivateKey, err = os.ReadFile(c.PrivateKey)
		if err != nil {
			return AppConfig{}, fmt.Errorf("read private key: %w", err)
		}
	}

	config.Tunnels, err = parseTunnelConfigs(c.Tunnels)
	if err != nil {
		return AppConfig{}, err
	}

	return config, nil
}

func parseTunnelConfigs(values []tunnelConfig) ([]TunnelSpec, error) {
	if len(values) == 0 {
		return nil, ErrNoTunnels
	}

	tunnels := make([]TunnelSpec, 0, len(values))
	for i, value := range values {
		tunnel, err := value.toTunnelSpec()
		if err != nil {
			return nil, fmt.Errorf("parse tunnel-%d: %w", i, err)
		}
		tunnels = append(tunnels, tunnel)
	}
	return tunnels, nil
}

func (c tunnelConfig) toTunnelSpec() (TunnelSpec, error) {
	local, err := ParseNetworkAddress(c.Local)
	if err != nil {
		return TunnelSpec{}, fmt.Errorf("parse local address: %w", err)
	}

	mode, err := ParseForwardMode(c.Mode)
	if err != nil {
		return TunnelSpec{}, err
	}
	if mode != ForwardDynamic && c.Remote == "" {
		return TunnelSpec{}, fmt.Errorf("remote address is required for %s mode", mode)
	}

	var remote NetworkAddress
	if c.Remote != "" {
		remote, err = ParseNetworkAddress(c.Remote)
		if err != nil {
			return TunnelSpec{}, fmt.Errorf("parse remote address: %w", err)
		}
	}

	return TunnelSpec{Local: local, Remote: remote, Mode: mode}, nil
}
