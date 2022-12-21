package refstr

import (
	"reflect"
	"testing"
)

func TestDecodeType(t *testing.T) {
	type Point struct{ X, Y float32 }

	tests := []struct {
		name     string
		typ      reflect.Type
		decode   string
		concrete any
		err      error
	}{{
		name:     "float32",
		typ:      TypeOf[float32](),
		decode:   "0.34",
		concrete: float32(0.34),
	}, {
		name:     "string",
		typ:      TypeOf[string](),
		decode:   "abc",
		concrete: string("abc"),
	}, {
		name:     "int",
		typ:      TypeOf[int](),
		decode:   "34",
		concrete: int(34),
	}, {
		name:     "*int",
		typ:      TypeOf[*int](),
		decode:   "34",
		concrete: int(34),
	}, {
		name:     "bool",
		typ:      TypeOf[bool](),
		decode:   "Y",
		concrete: bool(true),
	}, {
		name:     "[2]int",
		typ:      TypeOf[[2]int](),
		decode:   "3,4",
		concrete: [2]int{3, 4},
	}, {
		name:     "[]bool",
		typ:      TypeOf[[]bool](),
		decode:   "1 true, false,0",
		concrete: []bool{true, true, false, false},
	}, {
		name:     "map[string]int",
		typ:      TypeOf[map[string]int](),
		decode:   "a:2, b:5, c:6",
		concrete: map[string]int{"a": 2, "b": 5, "c": 6},
	}, {
		name:     "Point simple",
		typ:      TypeOf[Point](),
		decode:   "X:2, Y:5.4",
		concrete: Point{X: 2, Y: 5.4},
	}, {
		name:     "Point with braces",
		typ:      TypeOf[Point](),
		decode:   "{X:2 Y:5.4}",
		concrete: Point{X: 2, Y: 5.4},
	}}

	for _, test := range tests {
		val, err := DecodeType(test.typ, test.decode)

		if (err == nil) != (test.err == nil) {
			if err != nil {
				t.Errorf("[%s] Unexpected error during DecodeType: %v", test.name, err)
			} else {
				t.Errorf("[%s] Expecting error during DecodeType but none were returned: %v", test.name, test.err)
			}

			continue
		}

		concrete := Concrete(val).Interface()
		if !StringEqual(concrete, test.concrete) {
			t.Errorf("[%s] Expected %+v but got %+v", test.name, test.concrete, concrete)
		}
	}
}
