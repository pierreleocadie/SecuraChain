package network

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

func TestFilterOutPrivateAddrs(t *testing.T) {
	// Test case 1: Empty input
	addrs := []ma.Multiaddr{}
	expected := []ma.Multiaddr{}
	result := FilterOutPrivateAddrs(addrs)
	if len(result) != len(expected) {
		t.Errorf("Expected %d addresses, but got %d", len(expected), len(result))
	}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Expected address %s, but got %s", expected[i], result[i])
		}
	}

	// Test case 2: Only private addresses and localhost
	addrs = []ma.Multiaddr{
		ma.StringCast("/ip4/192.168.0.1/tcp/8080"),
		ma.StringCast("/ip6/fe80::1/tcp/8080"),
		ma.StringCast("/ip6/::1/tcp/8080"),
		ma.StringCast("/ip4/172.20.10.5/tcp/8080"),
		ma.StringCast("/ip4/127.0.0.1/tcp/8080"),
	}
	expected = []ma.Multiaddr{}
	result = FilterOutPrivateAddrs(addrs)
	if len(result) != len(expected) {
		t.Errorf("Expected %d addresses, but got %d", len(expected), len(result))
	}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Expected address %s, but got %s", expected[i], result[i])
		}
	}

	// Test case 3: Mix of private and public addresses
	addrs = []ma.Multiaddr{
		ma.StringCast("/ip4/192.168.0.1/tcp/8080"),
		ma.StringCast("/ip4/8.8.8.8/tcp/8080"),
		ma.StringCast("/ip6/fe80::1/tcp/8080"),
		ma.StringCast("/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/tcp/8080"),
	}
	expected = []ma.Multiaddr{
		ma.StringCast("/ip4/8.8.8.8/tcp/8080"),
		ma.StringCast("/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/tcp/8080"),
	}
	result = FilterOutPrivateAddrs(addrs)
	if len(result) != len(expected) {
		t.Errorf("Expected %d addresses, but got %d", len(expected), len(result))
	}
	for i := range result {
		if result[i].String() != expected[i].String() {
			t.Errorf("Expected address %s, but got %s", expected[i], result[i])
		}
	}
}
