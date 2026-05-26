package devicepairing

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// generatePairingCode encodes a one-time pairing code.
//
// Layout: base64url( rawToken[32] || utf8(serverURL) )
//
// The app decodes this to extract the server URL (bytes 32+) and the opaque
// token (bytes 0–31), then posts the full code to POST /api/v1/device-pair.
func generatePairingCode(serverURL string) (code string, rawToken []byte, err error) {
	rawToken = make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, rawToken); err != nil {
		return "", nil, fmt.Errorf("generate pairing token: %w", err)
	}

	payload := append(rawToken, []byte(serverURL)...)
	code = base64.RawURLEncoding.EncodeToString(payload)
	return code, rawToken, nil
}
