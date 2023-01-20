package refstr

import (
	"fmt"
	"reflect"
)

var defaultDecoder = NewDecoder()

// Decodes the string and applies it to the given v. v must be a pointer.
func Decode(v any, s string) error {
	return defaultDecoder.Decode(v, s)
}

// Decodes a value of the given type from the given string and returns it.
func DecodeType(t reflect.Type, s string) (any, error) {
	return defaultDecoder.DecodeType(t, s)
}

// Parses the string into the given type.
func Parse(s string, rt reflect.Type) (reflect.Value, error) {
	return defaultDecoder.Parse(s, rt)
}

// Converts the value to the given type.
func Convert(v any, rt reflect.Type) (any, error) {
	return defaultDecoder.Convert(v, rt)
}

// Returns the reference to the default decoder to control the global decoding logic.
func GetDefaultDecoder() *Decoder {
	return &defaultDecoder
}

// Returns a pointer to the given value.
func Ptr[V any](value V) *V {
	return &value
}

// Returns the reflect.Type based on the type parameter.
func TypeOf[V any]() reflect.Type {
	return reflect.TypeOf((*V)(nil)).Elem()
}

// Returns whether the given value is nil
func IsNil(v any) bool {
	rv := Reflect(v)
	for {
		switch rv.Kind() {
		case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
			if rv.IsNil() {
				return true
			}
		default:
			return false
		}
		if IsPointing(rv) {
			rv = rv.Elem()
		} else {
			return false
		}
	}
	return false
}

// Returns whether the two values string representations are equal.
func StringEqual(a any, b any) bool {
	return ToString(a) == ToString(b)
}

// Converts a value to a string.
func ToString(a any) string {
	return fmt.Sprintf("%+v", a)
}

// Converts the value to reflect.Value or if it is already it returns it as-is.
func Reflect(v any) reflect.Value {
	if rv, ok := v.(reflect.Value); ok {
		return rv
	}
	return reflect.ValueOf(v)
}

// Returns the concrete (non-pointer, non-interface) value of the given value.
func Concrete(v any) reflect.Value {
	rv := Reflect(v)
	for IsPointing(rv) {
		rv = rv.Elem()
	}
	return rv
}

// Returns true if the value is a pointer or interface.
func IsPointing(rv reflect.Value) bool {
	return rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer
}

// Converts the type to the non-pointer type.
func ConcreteType(rt reflect.Type) reflect.Type {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt
}

// Determines the concrete kind of the given value.
func ConcreteKind(v any) reflect.Kind {
	return Concrete(v).Kind()
}

// Returns a pointer to a concrete value if the given value is a pointer.
func PointerMaybe(v any) reflect.Value {
	rv := Reflect(v)
	if IsPointing(rv) && IsPointing(rv.Elem()) {
		rv = rv.Elem()
	}
	return rv
}

// Returns a pointer to the given value.
func PointerTo(rv reflect.Value) reflect.Value {
	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)
	return ptr
}

// Initializes the value inside rv to a non-nil value based on the given type.
// If it can't be done false is returned, otherwise true.
func InitValue(rv reflect.Value, rt reflect.Type) bool {
	if !rv.IsValid() {
		return false
	}
	switch rt.Kind() {
	case reflect.Slice, reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Chan:
		if !rv.IsNil() {
			return true
		}
		if !rv.CanSet() {
			return false
		}
	}

	switch rt.Kind() {
	case reflect.Slice:
		rv.Set(reflect.MakeSlice(rt, 0, 0))
	case reflect.Chan:
		rv.Set(reflect.MakeChan(rt, 0))
	case reflect.Map:
		rv.Set(reflect.MakeMap(rt))
	case reflect.Pointer:
		ptr := reflect.New(rt.Elem())
		if !InitValue(ptr.Elem(), rt.Elem()) {
			return false
		}
		rv.Set(ptr)
		// default:
		// 	if !rv.CanSet() {
		// 		return false
		// 	}
		// 	ptr := reflect.New(rt)
		// 	rv.Set(ptr.Elem())
	}

	return true
}

// Initializes a new value of the given type.
func InitType(rt reflect.Type) reflect.Value {
	rv := reflect.New(rt)
	if !InitValue(rv.Elem(), rt) {
		return reflect.Value{}
	}
	return rv.Elem()
}

// Initializes the given value (pointer to a value) and returns the reflect.Value.
func Init(value any) reflect.Value {
	rv := Reflect(value)
	if !InitValue(rv, rv.Type()) {
		return reflect.Value{}
	}
	return rv
}

// Determines whether the given type is a method or function which takes no arguments and returns a single value.
func IsGetter(rt reflect.Type, forType reflect.Type) bool {
	if rt.Kind() != reflect.Func {
		return false
	}
	if forType == nil && rt.NumIn() != 0 {
		return false
	}
	if forType != nil && (rt.NumIn() != 1 || rt.In(0) != forType) {
		return false
	}
	return rt.NumOut() == 1
}

// Determines whether the given type is a method or function which takes one argument and returns no value or an error.
func IsSetter(rt reflect.Type, forType reflect.Type) bool {
	if rt.Kind() != reflect.Func {
		return false
	}
	if rt.NumOut() > 1 || (rt.NumOut() == 1 && !TypeOf[error]().AssignableTo(rt.Out(0))) {
		return false
	}
	if forType == nil && rt.NumIn() != 1 {
		return false
	}
	if forType != nil && (rt.NumIn() != 2 || rt.In(0) != forType) {
		return false
	}
	return true
}
