package hostaccess

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
