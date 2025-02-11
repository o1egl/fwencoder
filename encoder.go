package fwencoder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

type columnWidthMap map[string]uint64

func (c columnWidthMap) Set(name string, width uint64) {
	old, ok := c[name]
	if !ok {
		c[name], old = uint64(len([]rune(name))), uint64(len([]rune(name)))
	}

	if old < width {
		c[name] = width
	}
}

// Marshal returns the fixed width table data encoding of v
// If v is nil or not a pointer to slice of structs, Unmarshal returns an ErrIncorrectInputValue.
//
// By default, Marshal converts struct's field names to column names. This behavior could be
// overridden by `column` or `json` tags.
//
// To unmarshal raw data into a struct, Unmarshal tries to convert every column's data from string to
// Marshal converts base go types into their string representation (int, int8, int16, int32, int64, uint, uint8, uint16,
// uint32, uint64, float32, float64, string, bool, time.Time)
// It also supports slices and custom types by converting them to JSON.
//
// By default, time.RFC3339 is used to parse time.Time data. To override this behavior use `format` tag.
// For example:
//
//	type Person struct {
//	    Name     string
//	    BDate    time.Time `column:"Birthday" format:"2006/01/02"`
//	    Postcode int       `json:"Zip"`
//	}
func Marshal(v any) ([]byte, error) {
	buf := bytes.Buffer{}
	err := MarshalWriter(&buf, v)
	return buf.Bytes(), err
}

// MarshalWriter behaves the same as Marshal, but write data into io.Writer
func MarshalWriter(writer io.Writer, v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	sliceItemType := reflect.TypeOf(v)
	if sliceItemType != nil && sliceItemType.Kind() == reflect.Ptr {
		sliceItemType = sliceItemType.Elem()
	} else {
		return ErrIncorrectInputValue
	}

	if sliceItemType.Kind() == reflect.Slice {
		sliceItemType = sliceItemType.Elem()
	} else {
		return ErrIncorrectInputValue
	}

	slice := reflect.ValueOf(v)
	if slice.Kind() == reflect.Ptr {
		slice = slice.Elem()
	}

	sliceType := sliceItemType
	if sliceType.Kind() == reflect.Ptr {
		sliceType = sliceType.Elem()
	}

	if sliceType.Kind() != reflect.Struct {
		return ErrIncorrectInputValue
	}

	columnNames := getColumns(sliceType)
	columnWidthIndex, err := makeColumnWidthIndex(slice, columnNames)
	if err != nil {
		return err
	}

	if err := writeHeader(writer, columnNames, columnWidthIndex); err != nil {
		return err
	}

	return writeData(writer, slice, columnWidthIndex)
}

func writeData(writer io.Writer, slice reflect.Value, columnWidthIndex columnWidthMap) error {
	for i := range slice.Len() {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		fieldsCount := item.NumField()
		for fieldIndex := range fieldsCount {
			fieldValue := item.Field(fieldIndex)
			fieldInfo := item.Type().Field(fieldIndex)
			refName := getRefName(&fieldInfo)
			columnWidth := columnWidthIndex[refName]
			if err := writeValue(writer, fieldValue, &fieldInfo, columnWidth); err != nil {
				return err
			}
			if fieldIndex != fieldsCount-1 {
				if _, err := writer.Write([]byte(" ")); err != nil {
					return err
				}
			}
		}

		if i != slice.Len()-1 {
			if _, err := writer.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeHeader(writer io.Writer, columnNames []string, columnWidthIndex columnWidthMap) error {
	for i, c := range columnNames {
		if _, err := fmt.Fprintf(writer, "%-"+strconv.FormatUint(columnWidthIndex[c], 10)+"s", c); err != nil {
			return err
		}
		if i != len(columnNames)-1 {
			if _, err := writer.Write([]byte(" ")); err != nil {
				return err
			}
		}
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func makeColumnWidthIndex(slice reflect.Value, columnNames []string) (columnWidthMap, error) {
	columnWidthIndex := make(columnWidthMap, len(columnNames))
	for i := range slice.Len() {
		item := slice.Index(i)

		if item.Kind() == reflect.Ptr {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}

		fieldsCount := item.NumField()
		for fieldIndex := range fieldsCount {
			currentField := item.Field(fieldIndex)
			typeField := item.Type().Field(fieldIndex)
			refName := getRefName(&typeField)
			fieldLen, err := getFieldLen(currentField, &typeField)
			if err != nil {
				return nil, err
			}
			columnWidthIndex.Set(refName, fieldLen)
		}
	}
	return columnWidthIndex, nil
}

//nolint:gocyclo // it's ok
func writeValue(w io.Writer, value reflect.Value, field *reflect.StructField, width uint64) error {
	gap := strconv.FormatUint(width, 10)

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			for range width {
				if _, err := w.Write([]byte(" ")); err != nil {
					return err
				}
			}
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, err := fmt.Fprintf(w, "%-"+gap+"d", value.Int()); err != nil {
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, err := fmt.Fprintf(w, "%-"+gap+"d", value.Uint()); err != nil {
			return err
		}
	case reflect.Float32, reflect.Float64:
		if _, err := fmt.Fprintf(w, "%-"+gap+"g", value.Float()); err != nil {
			return err
		}
	case reflect.Bool:
		if value.Bool() {
			if _, err := fmt.Fprintf(w, "%-"+gap+"s", "true"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "%-"+gap+"s", "false"); err != nil {
				return err
			}
		}
	case reflect.String:
		if _, err := fmt.Fprintf(w, "%-"+gap+"s", value.String()); err != nil {
			return err
		}
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(time.Time{}) {
			timeFormat, ok := field.Tag.Lookup(format)
			if !ok {
				timeFormat = time.RFC3339
			}
			if _, err := fmt.Fprintf(w, "%-"+gap+"s", value.Interface().(time.Time).Format(timeFormat)); err != nil {
				return err
			}
			return nil
		}
		fallthrough
	default:
		b, err := json.Marshal(value.Interface())
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%-"+gap+"s", string(b)); err != nil {
			return err
		}
	}
	return nil
}

func getFieldLen(value reflect.Value, field *reflect.StructField) (uint64, error) {
	const (
		trueLen  = 4
		falseLen = 5
	)

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return 0, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(len(strconv.FormatInt(value.Int(), 10))), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint64(len(strconv.FormatUint(value.Uint(), 10))), nil
	case reflect.Float32, reflect.Float64:
		return uint64(len(strconv.FormatFloat(value.Float(), 'f', -1, 64))), nil
	case reflect.Bool:
		if value.Bool() {
			return trueLen, nil
		} else {
			return falseLen, nil
		}
	case reflect.String:
		return uint64(len(value.String())), nil
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(time.Time{}) {
			timeFormat, ok := field.Tag.Lookup(format)
			if !ok {
				timeFormat = time.RFC3339
			}
			return uint64(len(value.Interface().(time.Time).Format(timeFormat))), nil
		}
		fallthrough
	default:
		b, err := json.Marshal(value.Interface())
		if err != nil {
			return 0, err
		}
		return uint64(len(b)), nil
	}
}
