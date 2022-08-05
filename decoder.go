package fwencoder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	columnTagName = "column"
	jsonTagName   = "json"
	format        = "format"
)

type fwColumn struct {
	name  string
	start int
	end   int
}

var (
	// ErrIncorrectInputValue represents wrong input param
	ErrIncorrectInputValue = errors.New("value is not a pointer to slice of structs")
)

// Unmarshal parses the fixed width table data and stores the result in the value pointed to by v.
// If v is nil or not a pointer to slice of structs, Unmarshal returns an ErrIncorrectInputValue.
//
// To unmarshal raw data into a struct, Unmarshal tries to convert every column's data from string to
// supported types (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string, bool, time.Time).
// It also supports slices and custom types by reading them as JSON.
//
// By default Unmarshal tries to match column names to struct's field names. This behavior could be
// overridden by `column` or `json` tags.
//
// By default time.RFC3339 is used to parse time.Time data. To override this behavior use `format` tag.
// For example:
//
//     type Person struct {
//         Name     string
//         BDate    time.Time `column:"Birthday" format:"2006/01/02"`
//         Postcode int       `json:"Zip"`
//     }
//
func Unmarshal(data []byte, v interface{}) error {
	return UnmarshalReader(bytes.NewReader(data), v)
}

//nolint:gocyclo
// UnmarshalReader behaves the same as Unmarshal, but reads data from io.Reader
func UnmarshalReader(reader io.Reader, v interface{}) (err error) {
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

	slice.Set(slice.Slice(0, 0))

	sliceType := sliceItemType
	if sliceType.Kind() == reflect.Ptr {
		sliceType = sliceType.Elem()
	}

	if sliceType.Kind() != reflect.Struct {
		return ErrIncorrectInputValue
	}

	scanner := bufio.NewScanner(reader)
	columnNames := getColumns(sliceType)
	sort.Slice(columnNames, func(i, j int) bool {
		return len([]rune(columnNames[i])) > len([]rune(columnNames[j]))
	})
	fieldsIndex := make(map[string]string)
	isHeaderParsed := false
	lineNum := 0
	headersLength := 0
	columns := make([]fwColumn, 0, len(columnNames))

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lineRunes := []rune(line)
		if !isHeaderParsed {
			isHeaderParsed = true
			headersLength = len(lineRunes)
			columns, err = parseHeaders(line, columnNames)
			if err != nil {
				return err
			}
			continue
		}
		if len(lineRunes) != headersLength {
			return fmt.Errorf("wrong data length in line %d", lineNum)
		}

		for _, prnColumn := range columns {
			fieldsIndex[prnColumn.name] = string(lineRunes[prnColumn.start:prnColumn.end])
		}

		newItem, err := createObject(fieldsIndex, sliceType)
		if err != nil {
			return fmt.Errorf("error in line %d: %w", lineNum, err)
		}
		if sliceItemType.Kind() != reflect.Ptr {
			newItem = newItem.Elem()
		}
		slice.Set(reflect.Append(slice, newItem))
	}

	return nil
}

func getRefName(field reflect.StructField) string {
	if name, ok := field.Tag.Lookup(columnTagName); ok {
		return name
	}
	if name, ok := field.Tag.Lookup(jsonTagName); ok {
		return name
	}
	return field.Name
}

func createObject(fieldsIndex map[string]string, t reflect.Type) (reflect.Value, error) {
	sp := reflect.New(t)
	s := sp.Elem()
	fieldsCount := s.NumField()
	for fieldIndex := 0; fieldIndex < fieldsCount; fieldIndex++ {
		currentField := s.Field(fieldIndex)
		typeField := s.Type().Field(fieldIndex)
		refName := getRefName(typeField)

		rawValue, ok := fieldsIndex[refName]
		if !ok {
			continue
		}
		if err := setFieldValue(currentField, typeField, rawValue); err != nil {
			return s, err
		}
	}
	return sp, nil
}

//nolint:gocyclo,funlen
func setFieldValue(field reflect.Value, structField reflect.StructField, rawValue string) error {
	rawValue = strings.TrimSpace(rawValue)
	fieldKind := field.Type().Kind()
	isPointer := fieldKind == reflect.Ptr
	if isPointer {
		fieldKind = field.Type().Elem().Kind()
	}
	//nolint:dupl
	switch fieldKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(rawValue, 10, 0)
		if err != nil {
			return newCastingError(err, rawValue, structField)
		}
		if isPointer {
			v := reflect.New(field.Type().Elem())
			if v.Elem().OverflowInt(value) {
				return newOverflowError(value, structField)
			}
			v.Elem().SetInt(value)
			field.Set(v)
		} else {
			if field.OverflowInt(value) {
				return newOverflowError(value, structField)
			}
			field.SetInt(value)
		}
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return newCastingError(err, rawValue, structField)
		}
		if isPointer {
			v := reflect.New(field.Type().Elem())
			if v.Elem().OverflowFloat(value) {
				return newOverflowError(value, structField)
			}
			v.Elem().SetFloat(value)
			field.Set(v)
		} else {
			if field.OverflowFloat(value) {
				return newOverflowError(value, structField)
			}
			field.SetFloat(value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return newCastingError(err, rawValue, structField)
		}
		if isPointer {
			v := reflect.New(field.Type().Elem())
			if v.Elem().OverflowUint(value) {
				return newOverflowError(value, structField)
			}
			v.Elem().SetUint(value)
			field.Set(v)
		} else {
			if field.OverflowUint(value) {
				return newOverflowError(value, structField)
			}
			field.SetUint(value)
		}
	case reflect.String:
		if isPointer {
			field.Set(reflect.ValueOf(&rawValue))
		} else {
			field.SetString(rawValue)
		}
	case reflect.Bool:
		value, err := strconv.ParseBool(rawValue)
		if err != nil {
			return newCastingError(err, rawValue, structField)
		}
		if isPointer {
			field.Set(reflect.ValueOf(&value))
		} else {
			field.SetBool(value)
		}
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) || field.Type() == reflect.TypeOf(&time.Time{}) {
			timeFormat, ok := structField.Tag.Lookup(format)
			if !ok {
				timeFormat = time.RFC3339
			}
			t, err := time.Parse(timeFormat, rawValue)
			if err != nil {
				return newCastingError(err, rawValue, structField)
			}
			if isPointer {
				field.Set(reflect.ValueOf(&t))
			} else {
				field.Set(reflect.ValueOf(t))
			}
			return nil
		}
		fallthrough
	default:
		v := reflect.New(field.Type())
		err := json.Unmarshal([]byte(rawValue), v.Interface())
		if err != nil {
			return fmt.Errorf(`can't unmarshal '"%s" to %v: %w`, rawValue, field.Type(), err)
		}
		field.Set(v.Elem())
	}
	return nil
}

func newCastingError(err error, rawValue string, structField reflect.StructField) error {
	return fmt.Errorf(`filed casting "%s" to "%s:%v": %w`, rawValue, structField.Name, structField.Type, err)
}

func newOverflowError(value any, structField reflect.StructField) error {
	return fmt.Errorf(`value %v is too big for field %s:%v`, value, structField.Name, structField.Type)
}

func getColumns(sType reflect.Type) []string {
	fCount := sType.NumField()
	columnNames := make([]string, 0, fCount)
	for i := 0; i < fCount; i++ {
		field := sType.Field(i)
		column := getRefName(field)
		columnNames = append(columnNames, column)
	}
	return columnNames
}

func parseHeaders(headerLine string, columnNames []string) ([]fwColumn, error) {
	columns := make([]fwColumn, 0, len(columnNames))
	for i := 0; i < len(columnNames); i++ {
		colName := columnNames[i]
		re, err := regexp.Compile(fmt.Sprintf("(%s *)", colName))
		if err != nil {
			return nil, fmt.Errorf("%s column parsing error: %w", colName, err)
		}

		loc := re.FindStringIndex(headerLine)
		if loc == nil {
			continue
		}
		col := fwColumn{
			name:  colName,
			start: loc[0],
			end:   loc[1],
		}
		columns = append(columns, col)
	}
	return columns, nil
}
