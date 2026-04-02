//go:build test

package device

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/matryer/is"
)

func ptrStr(s string) *string { return &s }

func TestDevice_Update_Rename(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "old-name", DeviceType: DeviceTypeStatic}

	err := d.Update(ptrStr("new-name"), nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "new-name")
}

func TestDevice_Update_NameUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "original", DeviceType: DeviceTypeStatic}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_InvalidNameEmpty(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "original", DeviceType: DeviceTypeStatic}
	empty := ""

	err := d.Update(&empty, nil, nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original") // not mutated on validation error
}

func TestDevice_Update_InvalidNameTooLong(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "original", DeviceType: DeviceTypeStatic}
	long := string(make([]rune, 51))

	err := d.Update(&long, nil, nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_SetDeviceType(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}

	err := d.Update(nil, ptrStr("mobile"), nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.DeviceType, DeviceTypeMobile)
}

func TestDevice_Update_DeviceTypeUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeMobile}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.DeviceType, DeviceTypeMobile)
}

func TestDevice_Update_InvalidDeviceType(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}

	err := d.Update(nil, ptrStr("invalid"), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.DeviceType, DeviceTypeStatic) // not mutated
}

func TestDevice_Update_SetDescription(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}
	desc := ptrStr("my server")

	err := d.Update(nil, nil, &desc, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "my server")
}

func TestDevice_Update_ClearDescription(t *testing.T) {
	is := is.New(t)
	orig := "has a description"
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic, Description: &orig}
	var nilPtr *string

	err := d.Update(nil, nil, &nilPtr, nil, nil)

	is.NoErr(err)
	is.True(d.Description == nil)
}

func TestDevice_Update_DescriptionUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	orig := "keep me"
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic, Description: &orig}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "keep me")
}

func TestDevice_Update_DescriptionTooLong(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}
	long := string(make([]rune, 201))
	longPtr := &long

	err := d.Update(nil, nil, &longPtr, nil, nil)

	is.True(err != nil)
	is.True(d.Description == nil)
}

func TestDevice_Update_SetIcon(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}
	icon := ptrStr("IconRouter")

	err := d.Update(nil, nil, nil, &icon, nil)

	is.NoErr(err)
	is.True(d.Icon != nil)
	is.Equal(*d.Icon, "IconRouter")
}

func TestDevice_Update_ClearIcon(t *testing.T) {
	is := is.New(t)
	orig := "IconServer"
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic, Icon: &orig}
	var nilPtr *string

	err := d.Update(nil, nil, nil, &nilPtr, nil)

	is.NoErr(err)
	is.True(d.Icon == nil)
}

func TestDevice_Update_IconTooLong(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "d", DeviceType: DeviceTypeStatic}
	long := string(make([]rune, 81))
	longPtr := &long

	err := d.Update(nil, nil, nil, &longPtr, nil)

	is.True(err != nil)
	is.True(d.Icon == nil)
}

func TestDevice_Update_NameNotMutatedWhenLaterFieldInvalid(t *testing.T) {
	is := is.New(t)
	d := &Device{Name: "original", DeviceType: DeviceTypeStatic}

	err := d.Update(ptrStr("new-name"), ptrStr("robot"), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original")             // name must not be written on validation failure
	is.Equal(d.DeviceType, DeviceTypeStatic) // type must not be written either
}

func TestNewCreateDeviceParams(t *testing.T) {
	is := is.New(t)

	params, rawKey, err := NewCreateDeviceParams("test-device", auth.UserID(1))
	is.NoErr(err)
	is.Equal(params.Name, "test-device")
	is.True(params.KeyPrefix != "")
	is.True(params.KeyHash != "")
	is.True(len(rawKey) > len(APIKeyPrefix))
	is.Equal(rawKey[:len(APIKeyPrefix)], APIKeyPrefix)
}
