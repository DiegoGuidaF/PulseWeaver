package httpapi

// Endpoints
const LoginEndpoint = "/api/v1/auth/login"
const HeartbeatEndpoint = "/api/v1/heartbeat"
const DevicePairEndpoint = "/api/v1/device-pair"

// Headers
const SessionCookieName = "__Host-wdc_session"
const APIKeyHeaderName = "X-API-Key"
const XRealIP = "X-Real-IP"

// Security scopes
const CookieAuthScope = "cookieAuth"
const APIKeyAuthScope = "apiKeyAuth"
