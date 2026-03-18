//go:build test

package device

import (
	"testing"

	"github.com/matryer/is"
)

func TestNewCreateDeviceParams(t *testing.T) {
	is := is.New(t)

	params, rawKey, err := NewCreateDeviceParams("test-device")
	is.NoErr(err)
	is.Equal(params.Name, "test-device")
	is.True(params.KeyPrefix != "")
	is.True(params.KeyHash != "")
	is.True(len(rawKey) > len(APIKeyPrefix))
	is.Equal(rawKey[:len(APIKeyPrefix)], APIKeyPrefix)
}
