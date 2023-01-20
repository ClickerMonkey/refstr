package refstr

import (
	"testing"
)

type point struct{ X, Y int }

func (p *point) Set(v int) {
	p.X = v
	p.Y = v
}

func (p point) Sum() int {
	return p.X + p.Y
}

func TestGetValueNodes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		nodes []string
	}{{
		name:  "struct",
		value: point{},
		nodes: []string{"X", "Y", "Sum"},
	}, {
		name:  "*struct",
		value: &point{},
		nodes: []string{"X", "Y", "Set", "Sum"},
	}, {
		name:  "map",
		value: map[string]int{"A": 4},
		nodes: []string{"A"},
	}, {
		name:  "array",
		value: [2]int{3, 4},
		nodes: []string{"0", "1"},
	}, {
		name:  "slice",
		value: []int{3},
		nodes: []string{"0"},
	}}

	for _, test := range tests {
		nodes := GetValueNodes(test.value)

		if !StringEqual(nodes.KeyStrings(), test.nodes) {
			t.Errorf("[%s] expected nodes %v but got %v", test.name, test.nodes, nodes.KeyStrings())
		}
	}
}

func TestRefGet(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		path     []any
		expected any
		err      error
	}{{
		name:     "struct",
		value:    point{X: 5},
		path:     []any{"X"},
		expected: int(5),
	}, {
		name:     "map",
		value:    map[string]int{"A": 6},
		path:     []any{"A"},
		expected: int(6),
	}, {
		name:     "map deep",
		value:    map[string]map[string]float32{"A": {"B": 3.4}},
		path:     []any{"A", "B"},
		expected: float32(3.4),
	}, {
		name:     "slice",
		value:    []string{"A"},
		path:     []any{0},
		expected: string("A"),
	}, {
		name:     "array",
		value:    [2]string{"A", "B"},
		path:     []any{1},
		expected: string("B"),
	}, {
		name:     "deep",
		value:    map[string]*point{"A": {X: 56}},
		path:     []any{"A", "X"},
		expected: int(56),
	}, {
		name:     "getter",
		value:    &point{X: 1, Y: 2},
		path:     []any{"Sum"},
		expected: int(3),
	}}

	for _, test := range tests {
		ref := NewRef(test.value).Nexts(test.path)
		if ref == nil {
			t.Errorf("[%s] There was a problem getting the path %v", test.name, test.path)
			continue
		}
		actual, err := ref.Get()

		if (err == nil) != (test.err == nil) {
			if err != nil {
				t.Errorf("[%s] Unexpected error during Path.Get: %v", test.name, err)
			} else {
				t.Errorf("[%s] Expecting error during Path.Get but none were returned: %v", test.name, test.err)
			}

			continue
		}

		if !StringEqual(actual.Interface(), test.expected) {
			t.Errorf("[%s] expected value %v but got %v", test.name, test.expected, actual.Interface())
		}
	}
}

type name struct{ First, Last string }

func (n name) Full() string { return n.Last + ", " + n.First }

type person struct{ Name name }
type persons struct{ ByName map[string]person }

func TestExample(t *testing.T) {
	p := persons{}
	pref := NewRef(&p)
	// references John's name in the map, but none of it exists yet.
	johnName := pref.Nexts([]any{"ByName", "John", "Name"})
	// creates john in the map and sets his first name
	johnName.Next("First").Set("John")
	// sets his last name
	johnName.Next("Last").Set("Doe")
	// gets his full name from the method Full()
	full, _ := johnName.Next("Full").Get()

	if full.Interface().(string) != "Doe, John" {
		t.FailNow()
	}
}

func TestRefSet(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		path     []any
		getPath  []any
		set      any
		expected any
		err      error
	}{{
		name:     "struct",
		value:    &point{X: 5},
		set:      int(6),
		path:     []any{"X"},
		expected: int(6),
	}, {
		name:     "map",
		value:    &map[string]int{"A": 6},
		set:      int(7),
		path:     []any{"A"},
		expected: int(7),
	}, {
		name:     "map add",
		value:    &map[string]int{"A": 6},
		set:      int(6),
		path:     []any{"B"},
		expected: int(6),
	}, {
		name:     "map deep",
		value:    &map[string]map[string]float32{"A": {"B": 3.4}},
		set:      float32(3.6),
		path:     []any{"A", "B"},
		expected: float32(3.6),
	}, {
		name:     "map deep add",
		value:    &map[string]map[string]float32{"A": {"B": 3.4}},
		set:      float32(3.4),
		path:     []any{"A", "C"},
		expected: float32(3.4),
	}, {
		name:     "slice",
		value:    &[]string{"A"},
		set:      string("B"),
		path:     []any{0},
		expected: string("B"),
	}, {
		name:     "slice add",
		value:    &[]string{"A"},
		set:      string("B"),
		path:     []any{1},
		expected: string("B"),
	}, {
		name:     "array",
		value:    &[2]string{"A", "B"},
		set:      string("C"),
		path:     []any{"1"},
		expected: string("C"),
	}, {
		name:     "deep",
		value:    &map[string]*point{"A": {X: 56}},
		set:      int(108),
		path:     []any{"A", "X"},
		expected: int(108),
	}, {
		name:     "deep add",
		value:    &map[string]*point{"A": {X: 56}},
		set:      int(109),
		path:     []any{"B", "X"},
		expected: int(109),
	}, {
		name:     "setter",
		value:    &point{X: 56},
		set:      int(57),
		path:     []any{"Set"},
		getPath:  []any{"X"},
		expected: int(57),
	}}

	for _, test := range tests {
		ref := NewRef(test.value).Nexts(test.path)
		err := ref.Set(test.set)

		if (err == nil) != (test.err == nil) {
			if err != nil {
				t.Errorf("[%s] Unexpected error during Path.Set: %v", test.name, err)
			} else {
				t.Errorf("[%s] Expecting error during Path.Set but none were returned: %v", test.name, test.err)
			}

			continue
		}

		if test.getPath != nil {
			ref = NewRef(test.value).Nexts(test.getPath)
		}

		actual, err := ref.Get()

		if (err == nil) != (test.err == nil) {
			if err != nil {
				t.Errorf("[%s] Unexpected error during Path.Get: %v", test.name, err)
			} else {
				t.Errorf("[%s] Expecting error during Path.Get but none were returned: %v", test.name, test.err)
			}

			continue
		}

		if !StringEqual(actual.Interface(), test.expected) {
			t.Errorf("[%s] expected value %v but got %v", test.name, test.expected, actual.Interface())
		}
	}
}
