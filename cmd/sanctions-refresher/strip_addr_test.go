package main

import (
	"errors"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A dial or DNS failure names the upstream host and port; stripURL drops them too, not only the URL.
func TestStripURL_DropsDialAndDNSAddress(t *testing.T) {
	t.Parallel()
	refused := &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}
	cases := map[string]struct {
		err    error
		secret []string
		is     error
	}{
		"dial refused": {
			err: &url.Error{Op: "Get", URL: "https://list.example.invalid:8443/x?token=s3cr3t", Err: &net.OpError{
				Op: "dial", Net: "tcp", Addr: &net.TCPAddr{IP: net.IPv4(192, 0, 2, 7), Port: 8443}, Err: refused,
			}},
			secret: []string{"192.0.2.7", "8443", "s3cr3t", "list.example.invalid"},
			is:     syscall.ECONNREFUSED,
		},
		"dns not found": {
			err: &url.Error{Op: "Get", URL: "https://list.example.invalid/x?token=s3cr3t", Err: &net.OpError{
				Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: "list.example.invalid", IsNotFound: true},
			}},
			secret: []string{"list.example.invalid", "s3cr3t"},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := stripURL(c.err)
			require.Error(t, got)
			for _, s := range c.secret {
				assert.NotContains(t, got.Error(), s)
			}
			if c.is != nil {
				assert.True(t, errors.Is(got, c.is), "the cause stays reachable")
			}
		})
	}
	// Negative control: the raw error does carry the address, so the assertion above is not vacuous.
	assert.Contains(t, cases["dial refused"].err.Error(), "192.0.2.7")
}
