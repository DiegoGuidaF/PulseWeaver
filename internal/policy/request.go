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
	Headers    map[string][]string
}

func NewVerifyRequest(token string, clientIP netip.Addr, r *http.Request) VerifyRequest {
	return VerifyRequest{
		Token:      token,
		ClientIP:   clientIP,
		TargetHost: nilIfEmpty(r.Header.Get("X-Forwarded-Host")),
		TargetURI:  nilIfEmpty(r.Header.Get("X-Forwarded-Uri")),
		HTTPMethod: nilIfEmpty(r.Header.Get("X-Forwarded-Method")),
		XFFChain:   nilIfEmpty(r.Header.Get("X-Forwarded-For")),
		Headers:    enrichmentHeaders(r),
	}
}

var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"proxy-authorization": true,
}

func enrichmentHeaders(r *http.Request) map[string][]string {
	headers := make(map[string][]string, len(r.Header))
	for key, vals := range r.Header {
		if !sensitiveHeaders[strings.ToLower(key)] {
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
