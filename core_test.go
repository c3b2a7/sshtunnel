package main

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestParseNetworkAddress(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "host port", value: "example.com:22", want: "example.com:22"},
		{name: "trim space", value: " example.com:22 ", want: "example.com:22"},
		{name: "ipv4", value: "127.0.0.1:1080", want: "127.0.0.1:1080"},
		{name: "missing port", value: "example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := ParseNetworkAddress(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseNetworkAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && addr.String() != tt.want {
				t.Fatalf("ParseNetworkAddress().String() = %q, want %q", addr.String(), tt.want)
			}
		})
	}
}

func TestParseForwardMode(t *testing.T) {
	tests := []struct {
		value   string
		want    ForwardMode
		wantErr bool
	}{
		{value: "local", want: ForwardLocal},
		{value: " REMOTE ", want: ForwardRemote},
		{value: "dynamic", want: ForwardDynamic},
		{value: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			mode, err := ParseForwardMode(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseForwardMode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if mode != tt.want {
				t.Fatalf("ParseForwardMode() = %q, want %q", mode, tt.want)
			}
		})
	}
}

func TestServeTunnelClosesLocalListenerOnCancel(t *testing.T) {
	addr := freeTCPAddress(t)
	ctx, cancel := context.WithCancel(context.Background())

	err := ServeTunnel(ctx, nil, TunnelSpec{
		Local: addr,
		Mode:  ForwardLocal,
	})
	if err != nil {
		t.Fatalf("ServeTunnel() error = %v", err)
	}

	cancel()

	deadline := time.Now().Add(time.Second)
	for {
		listener, err := net.Listen("tcp", addr.String())
		if err == nil {
			_ = listener.Close()
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("listener was not closed after context cancellation: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func freeTCPAddress(t *testing.T) NetworkAddress {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on free port: %v", err)
	}
	defer listener.Close()

	addr, err := ParseNetworkAddress(listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener address: %v", err)
	}
	return addr
}
