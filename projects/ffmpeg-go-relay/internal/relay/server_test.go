package relay

import "testing"

func TestParseUpstream(t *testing.T) {
	cases := []struct {
		input      string
		wantAddr   string
		wantTLS    bool
		wantScheme string
	}{
		{"rtmp://example.com:1935/app/stream", "example.com:1935", false, "rtmp"},
		{"rtmp://example.com/app/stream", "example.com:1935", false, "rtmp"},
		{"rtmps://example.com/app/stream", "example.com:1935", true, "rtmps"},
		{"rtsp://example.com/stream", "example.com:554", false, "rtsp"},
		{"rtsps://example.com/stream", "example.com:554", true, "rtsps"},
		{"example.com:1234/app", "example.com:1234", false, "rtmp"},
		{"rtmp://[2001:db8::1]/app", "[2001:db8::1]:1935", false, "rtmp"},
	}

	for _, c := range cases {
		info, err := ParseUpstream(c.input)
		if err != nil {
			t.Fatalf("parse %s: %v", c.input, err)
		}
		if info.Address != c.wantAddr {
			t.Fatalf("parse %s address = %s, want %s", c.input, info.Address, c.wantAddr)
		}
		if info.UseTLS != c.wantTLS {
			t.Fatalf("parse %s tls = %t, want %t", c.input, info.UseTLS, c.wantTLS)
		}
		if info.Scheme != c.wantScheme {
			t.Fatalf("parse %s scheme = %s, want %s", c.input, info.Scheme, c.wantScheme)
		}
	}
}

func TestParseUpstreamRejectsUnsupportedScheme(t *testing.T) {
	if _, err := ParseUpstream("http://example.com/stream"); err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
}

func TestExtractIP(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"192.168.0.1:1234", "192.168.0.1"},
		{"[2001:db8::1]:1935", "2001:db8::1"},
		{"example.com:443", "example.com"},
		{"justhost", "justhost"},
	}

	for _, c := range cases {
		if got := extractIP(c.input); got != c.want {
			t.Fatalf("extract %s = %s, want %s", c.input, got, c.want)
		}
	}
}
