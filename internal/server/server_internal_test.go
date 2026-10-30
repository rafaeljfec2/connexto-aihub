package server

import (
	"net"
	"testing"
)

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name string
		addr net.Addr
		want bool
	}{
		{name: "ipv4 loopback", addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, want: true},
		{name: "ipv6 loopback", addr: &net.TCPAddr{IP: net.ParseIP("::1")}, want: true},
		{name: "unspecified ipv4", addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.0")}, want: false},
		{name: "unspecified ipv6", addr: &net.TCPAddr{IP: net.ParseIP("::")}, want: false},
		{name: "private address", addr: &net.TCPAddr{IP: net.ParseIP("192.168.0.10")}, want: false},
		{name: "nil address", addr: nil, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isLoopback(test.addr); got != test.want {
				t.Fatalf("isLoopback(%v) = %v, want %v", test.addr, got, test.want)
			}
		})
	}
}
