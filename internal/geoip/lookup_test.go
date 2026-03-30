//go:build test

package geoip

import (
	"testing"

	"github.com/matryer/is"
)

func TestResult_IsEmpty(t *testing.T) {
	is := is.New(t)

	// Zero value is empty.
	is.True(Result{}.IsEmpty())

	// Any non-zero field makes it non-empty.
	is.True(!Result{CountryCode: "US"}.IsEmpty())
	is.True(!Result{CountryName: "United States"}.IsEmpty())
	is.True(!Result{ContinentCode: "NA"}.IsEmpty())
	is.True(!Result{ASN: 15169}.IsEmpty())
	is.True(!Result{ASNOrg: "Google LLC"}.IsEmpty())
}

func TestLookup_Resolve_NilReceiver(t *testing.T) {
	is := is.New(t)

	// A nil *Lookup must not panic and must return empty Result.
	var l *Lookup
	r := l.Resolve("8.8.8.8")
	is.True(r.IsEmpty())
}

func TestLookup_Resolve_NoReaders(t *testing.T) {
	is := is.New(t)

	// A zero-value Lookup (no readers loaded) returns empty Result for any IP.
	l := &Lookup{}
	r := l.Resolve("8.8.8.8")
	is.True(r.IsEmpty())
}

func TestLookup_Resolve_InvalidIP(t *testing.T) {
	is := is.New(t)

	// Lookup with no readers set returns empty Result for any IP.
	l := &Lookup{}
	r := l.Resolve("not-an-ip")
	is.True(r.IsEmpty())
}

func TestLookup_Resolve_PrivateIPWithNoReaders(t *testing.T) {
	is := is.New(t)

	// Lookup with no readers set returns empty Result even for valid IPs.
	l := &Lookup{}
	r := l.Resolve("192.168.1.1")
	is.True(r.IsEmpty())
}

func TestLookup_Resolve_PublicIPWithNoReaders(t *testing.T) {
	is := is.New(t)

	l := &Lookup{}
	r := l.Resolve("8.8.8.8")
	is.True(r.IsEmpty())
}

func TestLookup_Close_Nil(t *testing.T) {
	is := is.New(t)

	// Close on nil must not panic.
	var l *Lookup
	err := l.Close()
	is.NoErr(err)
}

func TestLookup_Close_Twice(t *testing.T) {
	is := is.New(t)

	// Double-close must not panic.
	l := &Lookup{}
	is.NoErr(l.Close())
	is.NoErr(l.Close())
}

func TestLookup_ConcurrentResolveAndClose(t *testing.T) {
	// Exercises the atomic.Pointer path under the race detector.
	// Run with: go test -tags=test -race ./internal/geoip/...
	l := &Lookup{}

	const goroutines = 50
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < goroutines; i++ {
			go func() {
				_ = l.Resolve("8.8.8.8")
			}()
		}
	}()

	// Concurrent Close swaps the atomic pointer.
	_ = l.Close()
	<-done
}
