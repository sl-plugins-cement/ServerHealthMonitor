package collector

import "testing"

func TestUDPPortListening(t *testing.T) {
	ssOutput := `udp   UNCONN 0      0    0.0.0.0:7777   0.0.0.0:*
udp   UNCONN 0      0    [::]:7777   [::]:*
udp   UNCONN 0      0    127.0.0.1:5353   0.0.0.0:*
`
	cases := []struct {
		port int
		want bool
	}{
		{7777, true},  // IPv4 wildcard
		{777, false},  // must NOT match the 7777 substring
		{5353, true},  // specific local address
		{535, false},  // must NOT match substring of 5353
		{9999, false}, // not present
		{0, false},    // invalid port
	}
	for _, tc := range cases {
		if got := UDPPortListening(ssOutput, tc.port); got != tc.want {
			t.Errorf("UDPPortListening(port=%d) = %v, want %v", tc.port, got, tc.want)
		}
	}
}

func TestUDPPortListeningIPv6(t *testing.T) {
	ssOutput := `udp6  UNCONN 0      0    [::1]:7777   [::]:*
`
	if !UDPPortListening(ssOutput, 7777) {
		t.Error("expected [::1]:7777 to be detected")
	}
	if UDPPortListening(ssOutput, 777) {
		t.Error("must not match substring of ipv6 local port")
	}
}

func TestLocalAddrPort(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0:7777": "7777",
		"[::]:7777":     "7777",
		"127.0.0.1:80":  "80",
		"[::]:*":        "*",
		"garbage":       "",
	}
	for in, want := range cases {
		if got := localAddrPort(in); got != want {
			t.Errorf("localAddrPort(%q) = %q, want %q", in, got, want)
		}
	}
}