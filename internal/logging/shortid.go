package logging

import (
	"crypto/rand"
	"encoding/hex"
)

// NewShortID returns a 6-character lowercase hex string derived from 3 random bytes.
// It is suitable for use as a short correlation ID in log entries.
func NewShortID() string {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		// rand.Read should never fail on a healthy system; fall back to a fixed sentinel
		// so callers are never blocked on an error path.
		return "000000"
	}
	return hex.EncodeToString(b)
}
