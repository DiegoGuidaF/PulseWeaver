package hosts

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestValidateFQDN_Valid(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"elephant-turtle-dns.wally.mywire.org",
		"a.bc",
		"my-host.internal.corp",
		"host123.example.co.uk",
		"UPPER.CASE.COM",
		"example.com.", // trailing dot is accepted
	}
	for _, fqdn := range valid {
		t.Run(fqdn, func(t *testing.T) {
			is := is.New(t)
			is.NoErr(ValidateFQDN(fqdn))
		})
	}
}

func TestValidateFQDN_Invalid(t *testing.T) {
	invalid := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single_label", "localhost"},
		{"leading_hyphen", "-bad.example.com"},
		{"trailing_hyphen", "bad-.example.com"},
		{"label_too_long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com"}, // 66-char label
		{"has_underscore", "my_host.example.com"},
		{"has_space", "my host.example.com"},
		{"has_scheme", "https://example.com"},
		{"has_port", "example.com:443"},
		{"has_path", "example.com/path"},
		{"ip_address", "192.168.1.1"},
		{"empty_label", "host..example.com"},
		{"just_dot", "."},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			err := ValidateFQDN(tc.input)
			is.True(err != nil)
			is.True(errors.Is(err, ErrBadRequest))
		})
	}
}

func TestNormaliseFQDN(t *testing.T) {
	is := is.New(t)

	is.Equal(NormaliseFQDN("  Example.COM  "), "example.com")
	is.Equal(NormaliseFQDN("host.example.com."), "host.example.com")
	is.Equal(NormaliseFQDN("ALREADY.lower.com"), "already.lower.com")
}

func TestNormaliseHost(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare", "example.com", "example.com"},
		{"non_default_port", "example.com:8443", "example.com"},
		{"https_default_port", "example.com:443", "example.com"},
		{"http_default_port", "example.com:80", "example.com"},
		{"uppercase_with_port", "EXAMPLE.COM:8443", "example.com"},
		{"trailing_dot_with_port", "example.com.:8443", "example.com"},
		{"whitespace_with_port", "  example.com:8443  ", "example.com"},
		// Non-numeric/malformed authorities are left intact (and so fail to match any
		// bare-FQDN grant) rather than being silently truncated to a real host.
		{"non_numeric_port", "example.com:notaport", "example.com:notaport"},
		{"empty_port", "example.com:", "example.com:"},
		{"ipv6_literal_with_port", "[2001:db8::1]:8443", "2001:db8::1"},
		{"bare_ipv6", "::1", "::1"},
		{"too_many_colons", "a:b:c", "a:b:c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(NormaliseHost(tc.in), tc.want)
		})
	}
}
