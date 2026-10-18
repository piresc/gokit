package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/piresc/gokit/middleware"
	"github.com/piresc/gokit/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name    string
		remote  string
		xff     []string
		realIP  string
		trusted []netip.Prefix
		want    string
	}{
		{"direct client", "203.0.113.5:1234", nil, "", nil, "203.0.113.5"},
		{"untrusted remote ignores XFF", "203.0.113.5:1234", []string{"1.2.3.4"}, "1.2.3.4", nil, "203.0.113.5"},
		{"one proxy", "172.18.0.3:5000", []string{"198.51.100.7"}, "", nil, "198.51.100.7"},
		{"spoofed first entry", "172.18.0.3:5000", []string{"1.2.3.4, 198.51.100.7"}, "", nil, "198.51.100.7"},
		{"two proxies", "172.18.0.3:5000", []string{"198.51.100.7, 10.0.0.2"}, "", nil, "198.51.100.7"},
		{"multiple header lines", "172.18.0.3:5000", []string{"1.2.3.4", "198.51.100.7"}, "", nil, "198.51.100.7"},
		{"X-Real-IP fallback", "172.18.0.3:5000", nil, "198.51.100.7", nil, "198.51.100.7"},
		{"garbage XFF", "172.18.0.3:5000", []string{"not-an-ip"}, "", nil, "172.18.0.3"},
		{"all hops private", "172.18.0.3:5000", []string{"10.0.0.5"}, "", nil, "10.0.0.5"},
		{"ipv4-mapped remote", "[::ffff:203.0.113.5]:1234", []string{"1.2.3.4"}, "", nil, "203.0.113.5"},
		{"ipv6 remote", "[2001:db8::1]:1234", []string{"1.2.3.4"}, "", nil, "2001:db8::1"},
		{"ipv4-mapped XFF hop", "172.18.0.3:5000", []string{"::ffff:198.51.100.7"}, "", nil, "198.51.100.7"},
		{"X-Real-IP from untrusted remote", "203.0.113.5:1234", nil, "1.2.3.4", nil, "203.0.113.5"},
		{"valid then garbage", "172.18.0.3:5000", []string{"198.51.100.7, garbage"}, "", nil, "172.18.0.3"},
		{"trust none", "172.18.0.3:5000", []string{"198.51.100.7"}, "", []netip.Prefix{}, "172.18.0.3"},
		{"unparseable RemoteAddr", "weird", nil, "", nil, "weird"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trusted := tc.trusted
			if trusted == nil {
				trusted = ratelimit.DefaultTrustedProxies()
			}
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if tc.realIP != "" {
				r.Header.Set("X-Real-IP", tc.realIP)
			}
			assert.Equal(t, tc.want, middleware.ClientIP(r, trusted))
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	got, err := ratelimit.ParseTrustedProxies([]string{"10.1.0.0/16"})
	require.NoError(t, err)
	assert.Equal(t, []netip.Prefix{netip.MustParsePrefix("10.1.0.0/16")}, got)

	_, err = ratelimit.ParseTrustedProxies([]string{"nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")

	got, err = ratelimit.ParseTrustedProxies(nil)
	require.NoError(t, err)
	assert.Equal(t, ratelimit.DefaultTrustedProxies(), got)
	assert.Len(t, got, 6)

	got, err = ratelimit.ParseTrustedProxies([]string{"none"})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}
