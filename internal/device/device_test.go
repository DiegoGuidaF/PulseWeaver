//go:build test

package device_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

func TestDevice_Update_Rename(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "old-name"}

	err := d.Update(new("new-name"), nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "new-name")
}

func TestDevice_Update_NameUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original"}

	err := d.Update(nil, nil, nil, nil)

	is.NoErr(err)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_InvalidNameEmpty(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original"}
	err := d.Update(new(""), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original") // not mutated on validation error
}

func TestDevice_Update_InvalidNameTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original"}
	err := d.Update(new(string(make([]rune, 51))), nil, nil, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original")
}

func TestDevice_Update_SetDescription(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d"}
	desc := new("my server")

	err := d.Update(nil, &desc, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "my server")
}

func TestDevice_Update_ClearDescription(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", Description: new("has a description")}
	var nilPtr *string

	err := d.Update(nil, &nilPtr, nil, nil)

	is.NoErr(err)
	is.True(d.Description == nil)
}

func TestDevice_Update_DescriptionUnchangedWhenNil(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", Description: new("keep me")}

	err := d.Update(nil, nil, nil, nil)

	is.NoErr(err)
	is.True(d.Description != nil)
	is.Equal(*d.Description, "keep me")
}

func TestDevice_Update_DescriptionTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d"}
	longPtr := new(string(make([]rune, 201)))

	err := d.Update(nil, &longPtr, nil, nil)

	is.True(err != nil)
	is.True(d.Description == nil)
}

func TestDevice_Update_SetIcon(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d"}
	icon := new("IconRouter")

	err := d.Update(nil, nil, &icon, nil)

	is.NoErr(err)
	is.True(d.Icon != nil)
	is.Equal(*d.Icon, "IconRouter")
}

func TestDevice_Update_ClearIcon(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d", Icon: new("IconServer")}
	var nilPtr *string

	err := d.Update(nil, nil, &nilPtr, nil)

	is.NoErr(err)
	is.True(d.Icon == nil)
}

func TestDevice_Update_IconTooLong(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "d"}
	longPtr := new(string(make([]rune, 81)))

	err := d.Update(nil, nil, &longPtr, nil)

	is.True(err != nil)
	is.True(d.Icon == nil)
}

func TestDevice_Update_NameNotMutatedWhenLaterFieldInvalid(t *testing.T) {
	is := is.New(t)
	d := &device.Device{Name: "original"}
	longIcon := new(string(make([]rune, 81)))

	err := d.Update(new("new-name"), nil, &longIcon, nil)

	is.True(err != nil)
	is.Equal(d.Name, "original") // name must not be written on validation failure
}
