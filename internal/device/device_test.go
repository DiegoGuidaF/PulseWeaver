package device

import (
	"testing"

	"github.com/matryer/is"
)

func TestNewDevice(t *testing.T) {
	is := is.New(t)

	device := NewDevice("test-device")
	is.True(device != nil)
	is.Equal(device.Name, "test-device")
	is.True(!device.CreatedAt.IsZero())
}
