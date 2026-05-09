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

func TestRelayCopiesBothDirections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	leftA, leftB := net.Pipe()
	rightA, rightB := net.Pipe()
	defer leftA.Close()
	defer leftB.Close()
	defer rightA.Close()
	defer rightB.Close()

	done := make(chan error, 1)
	go func() {
		done <- relay(ctx, leftA, rightA)
	}()

	if _, err := leftB.Write([]byte("from-left")); err != nil {
		t.Fatalf("write left side: %v", err)
	}
	buf := make([]byte, len("from-left"))
	if _, err := rightB.Read(buf); err != nil {
		t.Fatalf("read right side: %v", err)
	}
	if string(buf) != "from-left" {
		t.Fatalf("relay left->right = %q, want %q", string(buf), "from-left")
	}

	if _, err := rightB.Write([]byte("from-right")); err != nil {
		t.Fatalf("write right side: %v", err)
	}
	buf = make([]byte, len("from-right"))
	if _, err := leftB.Read(buf); err != nil {
		t.Fatalf("read left side: %v", err)
	}
	if string(buf) != "from-right" {
		t.Fatalf("relay right->left = %q, want %q", string(buf), "from-right")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("relay() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("relay() did not stop after context cancellation")
	}
}
