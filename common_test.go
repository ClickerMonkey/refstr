package refstr

import (
	"reflect"
	"testing"
)

func TestNewConcrete(t *testing.T) {
	type Point struct{ X, Y float32 }

	tests := []struct {
		name     string
		typ      reflect.Type
		concrete any
		isZero   bool
	}{{
		name:     "float32",
		typ:      TypeOf[float32](),
		concrete: float32(0),
	}, {
		name:     "*int",
		typ:      TypeOf[*int](),
		concrete: int(0),
	}, {
		name:     "**int",
		typ:      TypeOf[**int](),
		concrete: int(0),
	}, {
		name:     "Point",
		typ:      TypeOf[Point](),
		concrete: Point{},
	}, {
		name:     "*Point",
		typ:      TypeOf[*Point](),
		concrete: Point{},
	}}

	for _, test := range tests {
		concrete := InitType(test.typ)

		if test.isZero != !concrete.IsValid() {
			t.Errorf("[%s] Expected zero %v but got %v", test.name, test.isZero, !concrete.IsValid())
		} else {
			concreteValue := Concrete(concrete).Interface()

			if !StringEqual(concreteValue, test.concrete) {
				t.Errorf("[%s] Expected %+v but got %+v", test.name, test.concrete, concreteValue)
			}
		}

	}
}
