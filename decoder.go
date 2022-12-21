package refstr

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var ErrDecodeInvalid = errors.New("error decoding into given value - it must be a pointer and a supported string type")

// A custom parser for a specified type.
type Parser func(s string) (any, error)

// A decoder converts a string to a desired type.
type Decoder struct {
	Slice   Multi
	Array   Multi
	Map     Multi
	Struct  Multi
	Parsers map[reflect.Type]Parser
	Int     func(string, int) (int64, error)
	Uint    func(string, int) (uint64, error)
	Float   func(string, int) (float64, error)
	Complex func(string, int) (complex128, error)
	Trues   map[string]struct{}
	Falses  map[string]struct{}
}

// A type for controlling the parsing of multi-value types.
type Multi struct {
	Start          string
	ValueSeparator *regexp.Regexp
	KeySeparator   *regexp.Regexp
	End            string
	Strict         bool
}

// Converts the given string to a slice of strings based on the Multi options.
func (m Multi) Values(s string, max int) ([]string, error) {
	if m.Strict && (!strings.HasPrefix(s, m.Start) || !strings.HasSuffix(s, m.End)) {
		return nil, fmt.Errorf("error parsing multi-valued value with start '%s', end '%s' and value '%s'", m.Start, m.End, s)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(s, m.Start), m.End)
	values := m.ValueSeparator.Split(inner, max)
	return values, nil
}

// Converts the given string to a slice of key-value pairs based on the Multi options.
func (m Multi) KeyValues(s string, max int) ([][2]string, error) {
	entries, err := m.Values(s, max)
	if err != nil {
		return nil, err
	}
	keyValues := make([][2]string, len(entries))
	for i, entry := range entries {
		keyValue := m.KeySeparator.Split(entry, 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("error parsing key & value from '%s'", entry)
		}
		keyValues[i] = [2]string{keyValue[0], keyValue[1]}
	}
	return keyValues, nil
}

// Creates a new decoder with the default settings.
func NewDecoder() Decoder {
	vs := regexp.MustCompile(`\s*[\s,|]+\s*`)

	return Decoder{
		Slice:   Multi{Start: "[", ValueSeparator: vs, End: "]"},
		Array:   Multi{Start: "[", ValueSeparator: vs, End: "]"},
		Map:     Multi{Start: "map[", ValueSeparator: vs, KeySeparator: regexp.MustCompile(`:`), End: "]"},
		Struct:  Multi{Start: "{", ValueSeparator: vs, KeySeparator: regexp.MustCompile(`:`), End: "}"},
		Parsers: make(map[reflect.Type]Parser),
		Int:     func(s string, bits int) (int64, error) { return strconv.ParseInt(s, 10, bits) },
		Uint:    func(s string, bits int) (uint64, error) { return strconv.ParseUint(s, 10, bits) },
		Float:   func(s string, bits int) (float64, error) { return strconv.ParseFloat(s, bits) },
		Complex: func(s string, bits int) (complex128, error) { return strconv.ParseComplex(s, bits) },
		Trues: map[string]struct{}{
			"true": {},
			"t":    {},
			"yes":  {},
			"ya":   {},
			"y":    {},
			"si":   {},
			"1":    {},
			"x":    {},
		},
		Falses: map[string]struct{}{
			"false": {},
			"f":     {},
			"no":    {},
			"n":     {},
			"0":     {},
			"":      {},
		},
	}
}

var kindBits map[reflect.Kind]int = map[reflect.Kind]int{
	reflect.Complex128: 128,
	reflect.Complex64:  64,
	reflect.Float32:    32,
	reflect.Float64:    64,
	reflect.Int:        64,
	reflect.Int16:      16,
	reflect.Int32:      32,
	reflect.Int64:      64,
	reflect.Int8:       8,
	reflect.Uint:       64,
	reflect.Uint16:     16,
	reflect.Uint32:     32,
	reflect.Uint64:     64,
	reflect.Uint8:      8,
	reflect.Uintptr:    64,
}

// Decodes the string and applies it to the given v. v must be a pointer.
func (d Decoder) Decode(v any, s string) error {
	val := Init(v)
	if !val.IsValid() || val.Kind() != reflect.Pointer {
		return ErrDecodeInvalid
	}
	parsed, err := d.Parse(s, val.Type().Elem())
	if err != nil {
		return err
	}
	val.Elem().Set(parsed)
	return nil
}

// Parses the string into the given type.
func (d Decoder) Parse(s string, rt reflect.Type) (reflect.Value, error) {
	val := InitType(rt)
	concrete := Concrete(val)

	ptrMaybe := PointerMaybe(val).Interface()
	if unmarshaller, ok := ptrMaybe.(encoding.TextUnmarshaler); ok {
		err := unmarshaller.UnmarshalText([]byte(s))
		if err != nil {
			return val, fmt.Errorf("error unmarshalling text '%s': %w", s, err)
		}
		return val, nil
	}

	if parser, exists := d.Parsers[concrete.Type()]; exists {
		parsed, err := parser(s)
		if err != nil {
			return val, fmt.Errorf("error with custom parsing '%s': %w", s, err)
		}
		concrete.Set(reflect.ValueOf(parsed))
		return val, nil
	}

	k := concrete.Kind()
	switch k {
	case reflect.Bool:
		lower := strings.ToLower(s)
		if _, isTrue := d.Trues[lower]; isTrue {
			concrete.SetBool(true)
			return val, nil
		}
		if _, isFalse := d.Falses[lower]; isFalse {
			concrete.SetBool(false)
			return val, nil
		}
		return val, fmt.Errorf("error parsing '%s' as bool", s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := d.Int(s, kindBits[k])
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Kind(), err)
		}
		concrete.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := d.Uint(s, kindBits[k])
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Kind(), err)
		}
		concrete.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := d.Float(s, kindBits[k])
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Kind(), err)
		}
		concrete.SetFloat(parsed)
	case reflect.Complex64, reflect.Complex128:
		parsed, err := d.Complex(s, kindBits[k])
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Kind(), err)
		}
		concrete.SetComplex(parsed)
	case reflect.String:
		concrete.SetString(s)
	case reflect.Array:
		elements, err := d.Array.Values(s, concrete.Len())
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Type(), err)
		}
		elementType := concrete.Type().Elem()
		for i, elementString := range elements {
			element, err := d.Parse(elementString, elementType)
			if err != nil {
				return val, fmt.Errorf("error parsing '%s' as %v: %w", elementString, elementType, err)
			}
			concrete.Index(i).Set(element)
		}
	case reflect.Slice:
		if _, isBytes := concrete.Interface().([]byte); isBytes {
			concrete.SetBytes([]byte(s))
			return val, nil
		}

		elements, err := d.Slice.Values(s, -1)
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Type(), err)
		}
		elementType := concrete.Type().Elem()
		for _, elementString := range elements {
			element, err := d.Parse(elementString, elementType)
			if err != nil {
				return val, fmt.Errorf("error parsing '%s' as %v: %w", elementString, elementType, err)
			}
			concrete.Set(reflect.Append(concrete, element))
		}
	case reflect.Map:
		keyValues, err := d.Map.KeyValues(s, -1)
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Type(), err)
		}
		keyType := concrete.Type().Key()
		valueType := concrete.Type().Elem()

		for _, keyValue := range keyValues {
			key, err := d.Parse(keyValue[0], keyType)
			if err != nil {
				return val, fmt.Errorf("error parsing map key '%s' as %v: %w", keyValue[0], keyType, err)
			}
			value, err := d.Parse(keyValue[1], valueType)
			if err != nil {
				return val, fmt.Errorf("error parsing map value '%s' as %v: %w", keyValue[1], valueType, err)
			}
			concrete.SetMapIndex(key, value)
		}
	case reflect.Struct:
		keyValues, err := d.Struct.KeyValues(s, -1)
		if err != nil {
			return val, fmt.Errorf("error parsing '%s' as %v: %w", s, concrete.Type(), err)
		}

		for _, keyValue := range keyValues {
			fieldName := keyValue[0]
			field := concrete.FieldByName(fieldName)
			if !field.IsValid() {
				return val, fmt.Errorf("error parsing '%s', unknown field '%s'", s, fieldName)
			}
			value, err := d.Parse(keyValue[1], field.Type())
			if err != nil {
				return val, fmt.Errorf("error parsing struct field '%s' with value '%s' as %v: %w", fieldName, keyValue[1], field.Type(), err)
			}
			field.Set(value)
		}
	default:
		return val, fmt.Errorf("unsupported kind %v", concrete.Type())
	}

	return val, nil
}

// Decodes a value of the given type from the given string and returns it.
func (d Decoder) DecodeType(t reflect.Type, s string) (any, error) {
	v := reflect.New(t)
	err := d.Decode(v, s)
	if err != nil {
		return nil, err
	}
	return v.Elem().Interface(), nil
}
