//go:build test

package device_test

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

func TestNewAddress_ValidIPv4(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  device.DeviceID
		ipAddress string
		wantIP    string
		wantErr   bool
	}{
		{
			name:      "valid IPv4",
			deviceID:  device.DeviceID(1),
			ipAddress: "192.168.1.100",
			wantIP:    "192.168.1.100",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 with port",
			deviceID:  device.DeviceID(1),
			ipAddress: "192.168.1.100:8080",
			wantIP:    "192.168.1.100",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 localhost",
			deviceID:  device.DeviceID(1),
			ipAddress: "127.0.0.1",
			wantIP:    "",
			wantErr:   true,
		},
		{
			name:      "valid IPv4 with port localhost",
			deviceID:  device.DeviceID(1),
			ipAddress: "127.0.0.1:3000",
			wantIP:    "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			params, err := device.NewCreateAddressParams(tt.deviceID, tt.ipAddress, netip.Addr{})
			if tt.wantErr {
				is.True(err != nil)
				return
			}
			is.NoErr(err)
			is.Equal(params.DeviceID, tt.deviceID)
			is.Equal(params.IP.String(), tt.wantIP)
		})
	}
}

func TestNewAddress_ValidIPv6(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  device.DeviceID
		ipAddress string
		wantIP    string
		wantErr   bool
	}{
		{
			name:      "valid IPv6",
			deviceID:  device.DeviceID(1),
			ipAddress: "2001:db8::1",
			wantIP:    "2001:db8::1",
			wantErr:   false,
		},
		{
			name:      "valid IPv6 with port",
			deviceID:  device.DeviceID(1),
			ipAddress: "[2001:db8::1]:8080",
			wantIP:    "2001:db8::1",
			wantErr:   false,
		},
		{
			name:      "valid IPv6 localhost",
			deviceID:  device.DeviceID(1),
			ipAddress: "::1",
			wantIP:    "",
			wantErr:   true,
		},
		{
			name:      "valid IPv6 localhost with port",
			deviceID:  device.DeviceID(1),
			ipAddress: "[::1]:3000",
			wantIP:    "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			params, err := device.NewCreateAddressParams(tt.deviceID, tt.ipAddress, netip.Addr{})
			if tt.wantErr {
				is.True(err != nil)
				return
			}
			is.NoErr(err)
			is.Equal(params.DeviceID, tt.deviceID)
			is.Equal(params.IP.String(), tt.wantIP)
		})
	}
}

func TestNewAddress_InvalidIP(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  device.DeviceID
		ipAddress string
		wantErr   error
	}{
		{
			name:      "invalid IP format",
			deviceID:  device.DeviceID(1),
			ipAddress: "not.an.ip.address",
			wantErr:   device.ErrInvalidIPFormat,
		},
		{
			name:      "empty string",
			deviceID:  device.DeviceID(1),
			ipAddress: "",
			wantErr:   device.ErrInvalidIPFormat,
		},
		{
			name:      "invalid IPv4",
			deviceID:  device.DeviceID(1),
			ipAddress: "999.999.999.999",
			wantErr:   device.ErrInvalidIPFormat,
		},
		{
			name:      "invalid IPv6",
			deviceID:  device.DeviceID(1),
			ipAddress: "gggg::",
			wantErr:   device.ErrInvalidIPFormat,
		},
		{
			name:      "malformed port",
			deviceID:  device.DeviceID(1),
			ipAddress: "192.168.1.100:invalid",
			wantErr:   device.ErrInvalidIPFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			_, err := device.NewCreateAddressParams(tt.deviceID, tt.ipAddress, netip.Addr{})
			is.True(err != nil)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestNewCreateAddressParams_InvalidDeviceIP(t *testing.T) {
	tests := []struct {
		name      string
		ipAddress string
	}{
		{
			name:      "loopback IPv4",
			ipAddress: "127.0.0.1",
		},
		{
			name:      "loopback IPv6",
			ipAddress: "::1",
		},
		{
			name:      "multicast IPv4",
			ipAddress: "224.0.0.1",
		},
		{
			name:      "multicast IPv6",
			ipAddress: "ff02::1",
		},
		{
			name:      "unspecified IPv4",
			ipAddress: "0.0.0.0",
		},
		{
			name:      "unspecified IPv6",
			ipAddress: "::",
		},
		{
			name:      "link-local unicast IPv4",
			ipAddress: "169.254.1.1",
		},
		{
			name:      "link-local unicast IPv6",
			ipAddress: "fe80::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			_, err := device.NewCreateAddressParams(device.DeviceID(1), tt.ipAddress, netip.Addr{})
			is.True(err != nil)
			is.True(errors.Is(err, device.ErrInvalidDeviceIP))
		})
	}
}

func TestParseAndValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ipInput string
		want    string
		wantErr error
	}{
		{
			name:    "valid IPv4",
			ipInput: "192.168.1.100",
			want:    "192.168.1.100",
			wantErr: nil,
		},
		{
			name:    "valid IPv4 with port",
			ipInput: "192.168.1.100:8080",
			want:    "192.168.1.100",
			wantErr: nil,
		},
		{
			name:    "valid IPv6",
			ipInput: "2001:db8::1",
			want:    "2001:db8::1",
			wantErr: nil,
		},
		{
			name:    "valid IPv6 with port",
			ipInput: "[2001:db8::1]:8080",
			want:    "2001:db8::1",
			wantErr: nil,
		},
		{
			name:    "invalid IP format",
			ipInput: "not.an.ip",
			want:    "",
			wantErr: device.ErrInvalidIPFormat,
		},
		{
			name:    "empty string",
			ipInput: "",
			want:    "",
			wantErr: device.ErrInvalidIPFormat,
		},
		{
			name:    "invalid IPv4",
			ipInput: "999.999.999.999",
			want:    "",
			wantErr: device.ErrInvalidIPFormat,
		},
		{
			name:    "invalid IPv6",
			ipInput: "gggg::",
			want:    "",
			wantErr: device.ErrInvalidIPFormat,
		},
		{
			name:    "malformed port",
			ipInput: "192.168.1.100:invalid",
			want:    "",
			wantErr: device.ErrInvalidIPFormat,
		},
		{
			name:    "IPv4 localhost",
			ipInput: "127.0.0.1",
			want:    "127.0.0.1",
			wantErr: nil,
		},
		{
			name:    "IPv6 localhost",
			ipInput: "::1",
			want:    "::1",
			wantErr: nil,
		},
		{
			name:    "IPv4 localhost with port",
			ipInput: "127.0.0.1:3000",
			want:    "127.0.0.1",
			wantErr: nil,
		},
		{
			name:    "IPv6 localhost with port",
			ipInput: "[::1]:3000",
			want:    "::1",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			got, err := device.ParseAndValidateIP(tt.ipInput)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
			} else {
				is.NoErr(err)
				is.Equal(got.String(), tt.want)
			}
		})
	}
}
