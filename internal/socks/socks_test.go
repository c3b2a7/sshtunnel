package socks

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestParseAddrStringRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		atyp  byte
	}{
		{name: "domain", value: "example.com:443", want: "example.com:443", atyp: AtypDomainName},
		{name: "ipv4", value: "127.0.0.1:1080", want: "127.0.0.1:1080", atyp: AtypIPv4},
		{name: "ipv6", value: "[2001:db8::1]:22", want: "[2001:db8::1]:22", atyp: AtypIPv6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := ParseAddr(tt.value)
			if addr == nil {
				t.Fatalf("ParseAddr(%q) = nil", tt.value)
			}
			if addr[0] != tt.atyp {
				t.Fatalf("address type = %d, want %d", addr[0], tt.atyp)
			}
			if got := addr.String(); got != tt.want {
				t.Fatalf("Addr.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		err  Error
		want string
	}{
		{err: ErrAddressNotSupported, want: "address type not supported (8)"},
		{err: ErrCommandNotSupported, want: "command not supported (7)"},
		{err: Error(99), want: "error: 99"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAddrRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"example.com",
		"example.com:not-a-port",
		"example.com:65536",
		strings.Repeat("a", 256) + ":80",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if addr := ParseAddr(value); addr != nil {
				t.Fatalf("ParseAddr(%q) = %v, want nil", value, addr)
			}
		})
	}
}

func TestSplitAddr(t *testing.T) {
	addr := ParseAddr("example.com:443")
	if addr == nil {
		t.Fatal("ParseAddr() = nil")
	}
	input := append(append([]byte{}, addr...), 0xde, 0xad)

	got := SplitAddr(input)
	if !bytes.Equal(got, addr) {
		t.Fatalf("SplitAddr() = %v, want %v", got, addr)
	}
	if len(got) != len(addr) {
		t.Fatalf("SplitAddr() length = %d, want %d", len(got), len(addr))
	}
}

func TestSplitAddrRejectsMalformedInput(t *testing.T) {
	tests := [][]byte{
		nil,
		{0xff},
		{AtypDomainName},
		{AtypDomainName, 3, 'a'},
		{AtypIPv4, 127, 0, 0, 1, 0},
		{AtypIPv6, 0, 0},
	}

	for _, value := range tests {
		if addr := SplitAddr(value); addr != nil {
			t.Fatalf("SplitAddr(%v) = %v, want nil", value, addr)
		}
	}
}

func TestReadAddr(t *testing.T) {
	want := ParseAddr("127.0.0.1:1080")
	if want == nil {
		t.Fatal("ParseAddr() = nil")
	}

	got, err := ReadAddr(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("ReadAddr() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ReadAddr() = %v, want %v", got, want)
	}
}

func TestReadAddrRejectsUnsupportedAddressType(t *testing.T) {
	_, err := ReadAddr(bytes.NewReader([]byte{0xff}))
	if !errors.Is(err, ErrAddressNotSupported) {
		t.Fatalf("ReadAddr() error = %v, want %v", err, ErrAddressNotSupported)
	}
}

func TestReadAddrRequiresLargeEnoughBuffer(t *testing.T) {
	_, err := readAddr(bytes.NewReader(ParseAddr("127.0.0.1:1080")), make([]byte, MaxAddrLen-1))
	if !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("readAddr() error = %v, want %v", err, io.ErrShortBuffer)
	}
}

func TestHandshakeConnect(t *testing.T) {
	target := ParseAddr("example.com:443")
	if target == nil {
		t.Fatal("ParseAddr() = nil")
	}
	rw := newHandshakeReadWriter(
		append([]byte{5, 1, 0, 5, CmdConnect, 0}, target...),
	)

	addr, err := Handshake(rw)
	if err != nil {
		t.Fatalf("Handshake() error = %v", err)
	}
	if !bytes.Equal(addr, target) {
		t.Fatalf("Handshake() addr = %v, want %v", addr, target)
	}

	wantReply := []byte{5, 0, 5, 0, 0, AtypIPv4, 0, 0, 0, 0, 0, 0}
	if got := rw.out.Bytes(); !bytes.Equal(got, wantReply) {
		t.Fatalf("Handshake() reply = %v, want %v", got, wantReply)
	}
}

func TestHandshakeRejectsUnsupportedCommand(t *testing.T) {
	target := ParseAddr("example.com:443")
	if target == nil {
		t.Fatal("ParseAddr() = nil")
	}
	rw := newHandshakeReadWriter(
		append([]byte{5, 1, 0, 5, CmdBind, 0}, target...),
	)

	_, err := Handshake(rw)
	if !errors.Is(err, ErrCommandNotSupported) {
		t.Fatalf("Handshake() error = %v, want %v", err, ErrCommandNotSupported)
	}
	if got, want := rw.out.Bytes(), []byte{5, 0}; !bytes.Equal(got, want) {
		t.Fatalf("Handshake() reply = %v, want %v", got, want)
	}
}

func TestHandshakeUDPAssociateRequiresEnabledUDP(t *testing.T) {
	target := ParseAddr("example.com:443")
	if target == nil {
		t.Fatal("ParseAddr() = nil")
	}
	rw := newHandshakeReadWriter(
		append([]byte{5, 1, 0, 5, CmdUDPAssociate, 0}, target...),
	)

	_, err := Handshake(rw)
	if !errors.Is(err, ErrCommandNotSupported) {
		t.Fatalf("Handshake() error = %v, want %v", err, ErrCommandNotSupported)
	}
}

type handshakeReadWriter struct {
	*bytes.Reader
	out bytes.Buffer
}

func newHandshakeReadWriter(input []byte) *handshakeReadWriter {
	return &handshakeReadWriter{Reader: bytes.NewReader(input)}
}

func (rw *handshakeReadWriter) Write(p []byte) (int, error) {
	return rw.out.Write(p)
}

var _ io.ReadWriter = (*handshakeReadWriter)(nil)
