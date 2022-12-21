package refstr

import "testing"

func TestPathNodes(t *testing.T) {
	type Point struct{ X, Y int }

	tests := []struct {
		name  string
		value any
		nodes []string
	}{{
		name:  "struct",
		value: Point{},
		nodes: []string{"X", "Y"},
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
		nodes := GetPathNodes(test.value)

		if !StringEqual(nodes, test.nodes) {
			t.Errorf("[%s] expected nodes %v but got %v", test.name, test.nodes, nodes)
		}
	}
}

func TestPathGet(t *testing.T) {
	type Point struct{ X, Y int }

	tests := []struct {
		name     string
		value    any
		path     []string
		expected any
		err      error
	}{{
		name:     "struct",
		value:    Point{X: 5},
		path:     []string{"X"},
		expected: int(5),
	}, {
		name:     "map",
		value:    map[string]int{"A": 6},
		path:     []string{"A"},
		expected: int(6),
	}, {
		name:     "map deep",
		value:    map[string]map[string]float32{"A": map[string]float32{"B": 3.4}},
		path:     []string{"A", "B"},
		expected: float32(3.4),
	}, {
		name:     "slice",
		value:    []string{"A"},
		path:     []string{"0"},
		expected: string("A"),
	}, {
		name:     "array",
		value:    [2]string{"A", "B"},
		path:     []string{"1"},
		expected: string("B"),
	}, {
		name:     "deep",
		value:    map[string]*Point{"A": &Point{X: 56}},
		path:     []string{"A", "X"},
		expected: int(56),
	}}

	for _, test := range tests {
		actual, err := Path(test.path).Get(test.value)

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

func TestPathSet(t *testing.T) {
	type Point struct{ X, Y int }

	tests := []struct {
		name     string
		value    any
		path     []string
		set      any
		expected any
		err      error
	}{{
		name:     "struct",
		value:    &Point{X: 5},
		set:      int(6),
		path:     []string{"X"},
		expected: int(6),
		// }, {
		// 	name:     "map",
		// 	value:    &map[string]int{"A": 6},
		// 	set:      int(7),
		// 	path:     []string{"A"},
		// 	expected: int(7),
		// }, {
		// 	name:     "map deep",
		// 	value:    &map[string]map[string]float32{"A": map[string]float32{"B": 3.4}},
		// 	set:      float32(3.6),
		// 	path:     []string{"A", "B"},
		// 	expected: float32(3.6),
	}, {
		name:     "slice",
		value:    &[]string{"A"},
		set:      string("B"),
		path:     []string{"0"},
		expected: string("B"),
	}, {
		name:     "array",
		value:    &[2]string{"A", "B"},
		set:      string("C"),
		path:     []string{"1"},
		expected: string("C"),
		// }, {
		// 	name:     "deep",
		// 	value:    &map[string]*Point{"A": &Point{X: 56}},
		// 	set:      int(108),
		// 	path:     []string{"A", "X"},
		// 	expected: int(108),
	}}

	for _, test := range tests {
		path := Path(test.path)
		err := path.Set(test.value, test.set)

		if (err == nil) != (test.err == nil) {
			if err != nil {
				t.Errorf("[%s] Unexpected error during Path.Set: %v", test.name, err)
			} else {
				t.Errorf("[%s] Expecting error during Path.Set but none were returned: %v", test.name, test.err)
			}

			continue
		}

		actual, err := path.Get(test.value)

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
