package subscription

import (
	"net/netip"
	"net/url"
	"testing"
)

func TestValidateRemoteURLRejectsPrivateAndReservedAddresses(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/sub",
		"http://10.0.0.1/sub",
		"http://172.16.0.1/sub",
		"http://192.168.1.1/sub",
		"http://100.64.0.1/sub",
		"http://192.0.2.1/sub",
		"http://[::1]/sub",
		"http://[fe80::1]/sub",
		"http://[2001:db8::1]/sub",
	}
	for _, raw := range blocked {
		t.Run(raw, func(t *testing.T) {
			parsed, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateRemoteURL(parsed); err == nil {
				t.Fatalf("expected %s to be rejected", raw)
			}
		})
	}
}

func TestValidateRemoteURLAllowsPublicAddresses(t *testing.T) {
	allowed := []string{
		"https://example.com/sub",
		"http://172.15.0.1/sub",
		"http://172.32.0.1/sub",
		"http://8.8.8.8/sub",
	}
	for _, raw := range allowed {
		t.Run(raw, func(t *testing.T) {
			parsed, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateRemoteURL(parsed); err != nil {
				t.Fatalf("expected %s to be allowed: %v", raw, err)
			}
		})
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8":     true,
		"1.1.1.1":     true,
		"127.0.0.1":   false,
		"192.168.1.1": false,
		"100.64.0.1":  false,
		"192.0.2.1":   false,
		"::1":         false,
		"fe80::1":     false,
		"2001:db8::1": false,
	}
	for raw, want := range tests {
		if got := isPublicIP(netip.MustParseAddr(raw)); got != want {
			t.Fatalf("isPublicIP(%s) = %v, want %v", raw, got, want)
		}
	}
}
