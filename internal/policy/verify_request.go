package policy

import "net/http"

type VerifyRequest struct {
	Token      string
	ClientIP   string
	TargetHost *string
	TargetURI  *string
	HTTPMethod *string
	XFFChain   *string
	Headers    map[string][]string
}

func NewVerifyRequest(token string, clientIP string, r *http.Request) *VerifyRequest {
	return &VerifyRequest{
		Token:      token,
		ClientIP:   clientIP,
		TargetHost: nilIfEmpty(r.Header.Get("X-Forwarded-Host")),
		TargetURI:  nilIfEmpty(r.Header.Get("X-Forwarded-Uri")),
		HTTPMethod: nilIfEmpty(r.Header.Get("X-Forwarded-Method")),
		XFFChain:   nilIfEmpty(r.Header.Get("X-Forwarded-For")),
		Headers:    enrichmentHeaders(r),
	}
}

func enrichmentHeaders(r *http.Request) map[string][]string {
	headers := make(map[string][]string)
	for _, key := range []string{"User-Agent", "Referer", "Accept-Language", "X-Real-IP", "CF-Connecting-IP", "X-Forwarded-Proto"} {
		if vals := r.Header.Values(key); len(vals) > 0 {
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
