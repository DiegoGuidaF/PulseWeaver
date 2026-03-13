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
		Headers: map[string][]string{
			"User-Agent":        r.Header.Values("User-Agent"),
			"Referer":           r.Header.Values("Referer"),
			"Accept-Language":   r.Header.Values("Accept-Language"),
			"X-Real-IP":         r.Header.Values("X-Real-IP"),
			"CF-Connecting-IP":  r.Header.Values("CF-Connecting-IP"),
			"X-Forwarded-Proto": r.Header.Values("X-Forwarded-Proto"),
		},
	}
}
