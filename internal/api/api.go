package api

// Endpoints
const LoginEndpoint = "/api/v1/auth/login"

// Headers
const SessionCookieName = "__Host-wdc_session"
const ApiKeyHeaderName = "X-API-Key"
const XForwardedFor = "X-Forwarded-For"

// Security scopes
const CookieAuthScope = "cookieAuth"
const ApiKeyAuthScope = "apiKeyAuth"
