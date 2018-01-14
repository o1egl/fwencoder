package fwencoder

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMarshalWriter(t *testing.T) {
	buf := &bytes.Buffer{}

	s := "Test String"
	bb := false
	i := int8(-2)
	ui := uint8(2)
	f := float32(1.5)
	d := time.Date(2017, 12, 27, 0, 0, 0, 0, time.UTC)

	obj := []TestStruct{{
		String:    "Test String",
		Bool:      true,
		Int:       -1,
		Int8:      -2,
		Int16:     -3,
		Int32:     -4,
		Int64:     -5,
		Uint:      1,
		Uint8:     2,
		Uint16:    3,
		Uint32:    4,
		Uint64:    5,
		Float32:   1.5,
		Float64:   2.5,
		Date:      time.Date(2017, 12, 27, 13, 48, 3, 0, time.UTC),
		Birthday:  time.Date(2017, 12, 27, 0, 0, 0, 0, time.UTC),
		PString:   &s,
		PBool:     &bb,
		PInt8:     &i,
		PUint8:    &ui,
		PFloat32:  &f,
		PBirthday: &d,
		JsonArr:   []int{1, 2, 3},
		JsonPtr:   &[]int{4, 5, 6},
	},
		{
			String:  "Another test string",
			JsonPtr: &[]int{4, 5, 6},
		}}

	if assert.NoError(t, MarshalWriter(buf, &obj)) {
		b, err := ioutil.ReadFile("./testdata/marshal.txt")
		if b[len(b)-1] == '\n' {
			b = b[:len(b)-1]
		}
		if assert.NoError(t, err) {
			assert.Equal(t, string(b), buf.String())
		}
	}
}

func TestMarshalPtr(t *testing.T) {
	buf := &bytes.Buffer{}

	s := "Test String"
	bb := false
	i := int8(-2)
	ui := uint8(2)
	f := float32(1.5)
	d := time.Date(2017, 12, 27, 0, 0, 0, 0, time.UTC)

	obj := []*TestStruct{{
		String:    "Test String",
		Bool:      true,
		Int:       -1,
		Int8:      -2,
		Int16:     -3,
		Int32:     -4,
		Int64:     -5,
		Uint:      1,
		Uint8:     2,
		Uint16:    3,
		Uint32:    4,
		Uint64:    5,
		Float32:   1.5,
		Float64:   2.5,
		Date:      time.Date(2017, 12, 27, 13, 48, 3, 0, time.UTC),
		Birthday:  time.Date(2017, 12, 27, 0, 0, 0, 0, time.UTC),
		PString:   &s,
		PBool:     &bb,
		PInt8:     &i,
		PUint8:    &ui,
		PFloat32:  &f,
		PBirthday: &d,
		JsonArr:   []int{1, 2, 3},
		JsonPtr:   &[]int{4, 5, 6},
	},
		{
			String:  "Another test string",
			JsonPtr: &[]int{4, 5, 6},
		}}

	if assert.NoError(t, MarshalWriter(buf, &obj)) {
		b, err := ioutil.ReadFile("./testdata/marshal.txt")
		if b[len(b)-1] == '\n' {
			b = b[:len(b)-1]
		}
		if assert.NoError(t, err) {
			assert.Equal(t, string(b), buf.String())
		}
	}
}

func TestMarshal_IncorrectInput(t *testing.T) {
	errs := []error{
		marshallErr(1),
		marshallErr(nil),
		marshallErr(new(string)),
		marshallErr(&([]int{})),
	}

	for _, err := range errs {
		assert.EqualError(t, err, ErrIncorrectInputValue.Error())
	}
}

func marshallErr(i interface{}) error {
	_, err := Marshal(i)
	return err
}
