//go:build test

package device_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

func TestDevice_Update_Rename(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "old-name", DeviceType: device.DeviceTypeStatic}

	err := d.Update(new("new-name"), nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "new-name")
}

func TestDevice_Update_NameUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original", DeviceType: device.DeviceTypeStatic}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_InvalidNameEmpty(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original", DeviceType: device.DeviceTypeStatic}
	err := d.Update(new(""), nil, nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original") // not mutated on validation error
}

func TestDevice_Update_InvalidNameTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original", DeviceType: device.DeviceTypeStatic}
	err := d.Update(new(string(make([]rune, 51))), nil, nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_SetDeviceType(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}

	err := d.Update(nil, new("mobile"), nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.DeviceType, device.DeviceTypeMobile)
}

func TestDevice_Update_DeviceTypeUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeMobile}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.DeviceType, device.DeviceTypeMobile)
}

func TestDevice_Update_InvalidDeviceType(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}

	err := d.Update(nil, new("invalid"), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.DeviceType, device.DeviceTypeStatic) // not mutated
}

func TestDevice_Update_SetDescription(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}
	desc := new("my server")

	err := d.Update(nil, nil, &desc, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "my server")
}

func TestDevice_Update_ClearDescription(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic, Description: new("has a description")}
	var nilPtr *string

	err := d.Update(nil, nil, &nilPtr, nil, nil)

	is.NoErr(err)
	is.True(d.Description == nil)
}

func TestDevice_Update_DescriptionUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic, Description: new("keep me")}

	err := d.Update(nil, nil, nil, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "keep me")
}

func TestDevice_Update_DescriptionTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}
	longPtr := new(string(make([]rune, 201)))

	err := d.Update(nil, nil, &longPtr, nil, nil)

	is.True(err != nil)
	is.True(d.Description == nil)
}

func TestDevice_Update_SetIcon(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}
	icon := new("IconRouter")

	err := d.Update(nil, nil, nil, &icon, nil)

	is.NoErr(err)
	is.True(d.Icon != nil)
	is.Equal(*d.Icon, "IconRouter")
}

func TestDevice_Update_ClearIcon(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic, Icon: new("IconServer")}
	var nilPtr *string

	err := d.Update(nil, nil, nil, &nilPtr, nil)

	is.NoErr(err)
	is.True(d.Icon == nil)
}

func TestDevice_Update_IconTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", DeviceType: device.DeviceTypeStatic}
	longPtr := new(string(make([]rune, 81)))

	err := d.Update(nil, nil, nil, &longPtr, nil)

	is.True(err != nil)
	is.True(d.Icon == nil)
}

func TestDevice_Update_NameNotMutatedWhenLaterFieldInvalid(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original", DeviceType: device.DeviceTypeStatic}

	err := d.Update(new("new-name"), new("robot"), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original")                    // name must not be written on validation failure
	is.Equal(d.DeviceType, device.DeviceTypeStatic) // type must not be written either
}
