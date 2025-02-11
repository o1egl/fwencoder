package fwencoder

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	String    string
	Bool      bool
	Int       int
	Int8      int8
	Int16     int16
	Int32     int32
	Int64     int64
	Uint      uint
	Uint8     uint8
	Uint16    uint16
	Uint32    uint32
	Uint64    uint64
	Float32   float32
	Float64   float64
	Date      time.Time `json:"Time"`
	Birthday  time.Time `column:"CustomDate" format:"02/01/2006"`
	PString   *string
	PBool     *bool
	PInt8     *int8
	PUint8    *uint8
	PFloat32  *float32
	PBirthday *time.Time `format:"02/01/2006"`
	Default   int
	JSONArr   []int
	JSONPtr   *[]int
}

func TestUnmarshal_Success(t *testing.T) {
	b, err := os.ReadFile("./testdata/correct_all_supported.txt")
	if assert.NoError(t, err) {
		s := "Test Ptr String"
		bb := false
		i := int8(15)
		ui := uint8(16)
		f := float32(15.5)
		d := time.Date(2017, 12, 28, 0, 0, 0, 0, time.UTC)

		expected := []TestStruct{{
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
			JSONArr:   []int{1, 2, 3},
			JSONPtr:   &[]int{4, 5, 6},
		}}

		var obtained []TestStruct
		if err := Unmarshal(b, &obtained); assert.NoError(t, err) {
			assert.Equal(t, expected, obtained)
		}
	}
}

func TestUnmarshal_Ptr_Success(t *testing.T) {
	b, err := os.ReadFile("./testdata/correct_all_supported.txt")
	if assert.NoError(t, err) {
		s := "Test Ptr String"
		bb := false
		i := int8(15)
		ui := uint8(16)
		f := float32(15.5)
		d := time.Date(2017, 12, 28, 0, 0, 0, 0, time.UTC)

		expected := []*TestStruct{{
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
			JSONArr:   []int{1, 2, 3},
			JSONPtr:   &[]int{4, 5, 6},
		}}

		var obtained []*TestStruct
		if err := Unmarshal(b, &obtained); assert.NoError(t, err) {
			assert.Equal(t, expected, obtained)
		}
	}
}

type BadData struct {
	Data  []byte
	Error string
}

func TestUnmarshal_Error(t *testing.T) {
	badData := []BadData{
		{
			Data: []byte(`
		Int
		5.3`),
			Error: `filed casting "5.3" to "Int:int"`,
		},
		{
			Data: []byte(`
		Bool
		5.3 `),
			Error: `filed casting "5.3" to "Bool:bool"`,
		},
		{
			Data: []byte(`
		Uint
		5.3 `),
			Error: `filed casting "5.3" to "Uint:uint"`,
		},
		{
			Data: []byte(`
		Float32
		hello  `),
			Error: `filed casting "hello" to "Float32:float32"`,
		},
		{
			Data: []byte(`
		Time
		5.3 `),
			Error: `filed casting "5.3" to "Date:time.Time"`,
		},
		{
			Data: []byte(`
		Int
		5`),
			Error: `wrong data length in line 2`,
		},
		{
			Data: []byte(`
		Int8
		5123`),
			Error: `is too big for field Int8:int8`,
		},
		{
			Data: []byte(`
		Uint8
		5123 `),
			Error: `is too big for field Uint8:uint8`,
		},
		{
			Data:  []byte(fmt.Sprintf(" %-309s\n%.0f", "Float32", math.MaxFloat64)),
			Error: `is too big for field Float32:float32`,
		},
	}
	for _, data := range badData {
		var obtained []TestStruct
		err := Unmarshal(data.Data[1:], &obtained)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), data.Error)
		}
	}
}

func TestIncorrectInput(t *testing.T) {
	errs := []error{
		Unmarshal(nil, 1),
		Unmarshal(nil, nil),
		Unmarshal(nil, new(string)),
		Unmarshal(nil, &([]int{})),
	}

	for _, err := range errs {
		require.EqualError(t, err, ErrIncorrectInputValue.Error())
	}

	type B struct {
		Int int `column:")Float32"`
	}

	type A struct {
		Float32 B
	}

	err := Unmarshal([]byte("Float32\nhello  "), &([]A{}))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "error in line 2: can't unmarshal")
	}

	err = Unmarshal([]byte("Float32\nhello  "), &([]B{}))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "error parsing regexp")
	}
}

func TestPtrFieldsOverflow(t *testing.T) {
	badData := []BadData{
		{
			Data: []byte(`
		Int8
		5123`),
			Error: `is too big for field Int8:*int8`,
		},
		{
			Data: []byte(`
		Uint8
		5123 `),
			Error: `is too big for field Uint8:*uint8`,
		},
		{
			Data:  []byte(fmt.Sprintf(" %-309s\n%.0f", "Float32", math.MaxFloat64)),
			Error: `is too big for field Float32:*float32`,
		},
	}

	// Pointer fields overflow
	type A struct {
		Int8    *int8
		Uint8   *uint8
		Float32 *float32
	}

	for _, data := range badData {
		var obtained []A
		err := Unmarshal(data.Data[1:], &obtained)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), data.Error)
		}
	}
}
