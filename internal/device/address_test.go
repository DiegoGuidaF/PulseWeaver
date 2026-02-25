package device

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestNewAddress_ValidIPv4(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  DeviceID
		ipAddress string
		wantIP    string
		wantErr   bool
	}{
		{
			name:      "valid IPv4",
			deviceID:  DeviceID(1),
			ipAddress: "192.168.1.100",
			wantIP:    "192.168.1.100",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 with port",
			deviceID:  DeviceID(1),
			ipAddress: "192.168.1.100:8080",
			wantIP:    "192.168.1.100",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 localhost",
			deviceID:  DeviceID(1),
			ipAddress: "127.0.0.1",
			wantIP:    "127.0.0.1",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 with port localhost",
			deviceID:  DeviceID(1),
			ipAddress: "127.0.0.1:3000",
			wantIP:    "127.0.0.1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			params, err := NewCreateAddressParams(tt.deviceID, tt.ipAddress)
			if tt.wantErr {
				is.True(err != nil)
				return
			}
			is.NoErr(err)
			is.True(params != nil)
			is.Equal(params.DeviceID, tt.deviceID)
			is.Equal(params.IP, tt.wantIP)
		})
	}
}

func TestNewAddress_ValidIPv6(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  DeviceID
		ipAddress string
		wantIP    string
		wantErr   bool
	}{
		{
			name:      "valid IPv6",
			deviceID:  DeviceID(1),
			ipAddress: "2001:db8::1",
			wantIP:    "2001:db8::1",
			wantErr:   false,
		},
		{
			name:      "valid IPv6 with port",
			deviceID:  DeviceID(1),
			ipAddress: "[2001:db8::1]:8080",
			wantIP:    "2001:db8::1",
			wantErr:   false,
		},
		{
			name:      "valid IPv6 localhost",
			deviceID:  DeviceID(1),
			ipAddress: "::1",
			wantIP:    "::1",
			wantErr:   false,
		},
		{
			name:      "valid IPv6 localhost with port",
			deviceID:  DeviceID(1),
			ipAddress: "[::1]:3000",
			wantIP:    "::1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			params, err := NewCreateAddressParams(tt.deviceID, tt.ipAddress)
			if tt.wantErr {
				is.True(err != nil)
				return
			}
			is.NoErr(err)
			is.True(params != nil)
			is.Equal(params.DeviceID, tt.deviceID)
			is.Equal(params.IP, tt.wantIP)
		})
	}
}

func TestNewAddress_InvalidIP(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  DeviceID
		ipAddress string
		wantErr   error
	}{
		{
			name:      "invalid IP format",
			deviceID:  DeviceID(1),
			ipAddress: "not.an.ip.address",
			wantErr:   ErrInvalidIPFormat,
		},
		{
			name:      "empty string",
			deviceID:  DeviceID(1),
			ipAddress: "",
			wantErr:   ErrInvalidIPFormat,
		},
		{
			name:      "invalid IPv4",
			deviceID:  DeviceID(1),
			ipAddress: "999.999.999.999",
			wantErr:   ErrInvalidIPFormat,
		},
		{
			name:      "invalid IPv6",
			deviceID:  DeviceID(1),
			ipAddress: "gggg::",
			wantErr:   ErrInvalidIPFormat,
		},
		{
			name:      "malformed port",
			deviceID:  DeviceID(1),
			ipAddress: "192.168.1.100:invalid",
			wantErr:   ErrInvalidIPFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			params, err := NewCreateAddressParams(tt.deviceID, tt.ipAddress)
			is.True(err != nil)
			is.True(errors.Is(err, tt.wantErr))
			is.True(params == nil)
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
			wantErr: ErrInvalidIPFormat,
		},
		{
			name:    "empty string",
			ipInput: "",
			want:    "",
			wantErr: ErrInvalidIPFormat,
		},
		{
			name:    "invalid IPv4",
			ipInput: "999.999.999.999",
			want:    "",
			wantErr: ErrInvalidIPFormat,
		},
		{
			name:    "invalid IPv6",
			ipInput: "gggg::",
			want:    "",
			wantErr: ErrInvalidIPFormat,
		},
		{
			name:    "malformed port",
			ipInput: "192.168.1.100:invalid",
			want:    "",
			wantErr: ErrInvalidIPFormat,
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
			got, err := parseAndValidateIP(tt.ipInput)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
			} else {
				is.NoErr(err)
				is.Equal(got, tt.want)
			}
		})
	}
}
