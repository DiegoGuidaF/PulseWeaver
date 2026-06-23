package policy

import (
	"net/http"
	"net/netip"
	"strings"
)

type VerifyRequest struct {
	Token      string
	ClientIP   netip.Addr
	TargetHost *string
	TargetURI  *string
	HTTPMethod *string
	XFFChain   *string
	// rawHeader is the request's header set, kept by reference (not cloned) so
	// the access-log header map can be materialized lazily and only at the width
	// the outcome warrants — full on allow, the minimal forwarding subset on deny.
	rawHeader http.Header
}

func NewVerifyRequest(token string, clientIP netip.Addr, r *http.Request) VerifyRequest {
	return VerifyRequest{
		Token:      token,
		ClientIP:   clientIP,
		TargetHost: nilIfEmpty(r.Header.Get("X-Forwarded-Host")),
		TargetURI:  nilIfEmpty(r.Header.Get("X-Forwarded-Uri")),
		HTTPMethod: nilIfEmpty(r.Header.Get("X-Forwarded-Method")),
		XFFChain:   nilIfEmpty(r.Header.Get("X-Forwarded-For")),
		rawHeader:  r.Header,
	}
}

var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"proxy-authorization": true,
}

// forwardingHeaderKeys is the allowlist retained on a deny: the proxy-supplied
// forwarding context, enough to attribute the request in the audit log without
// cloning the client's full header set. It is a fixed
// allowlist, so a deny row can never carry a sensitive header by construction.
var forwardingHeaderKeys = []string{
	"X-Forwarded-Host",
	"X-Forwarded-Uri",
	"X-Forwarded-Method",
	"X-Forwarded-Proto",
	"X-Forwarded-For",
	"X-Real-Ip",
	"User-Agent",
}

// fullEnrichmentHeaders copies every request header except the sensitive ones.
// Used for allowed (proxied) requests, where the full header set is the audit value.
func fullEnrichmentHeaders(h http.Header) map[string][]string {
	headers := make(map[string][]string, len(h))
	for key, vals := range h {
		if !sensitiveHeaders[strings.ToLower(key)] {
			headers[key] = vals
		}
	}
	return headers
}

// minimalEnrichmentHeaders retains only the forwarding-context allowlist. Used
// for denies, where the full client header set is not worth the per-request
// allocation churn under load (the deny path is the flood case).
func minimalEnrichmentHeaders(h http.Header) map[string][]string {
	headers := make(map[string][]string, len(forwardingHeaderKeys))
	for _, key := range forwardingHeaderKeys {
		if vals := h.Values(key); len(vals) > 0 {
			headers[key] = vals
		}
	}
	return headers
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
