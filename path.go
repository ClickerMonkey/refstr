package refstr

import (
	"fmt"
	"reflect"
	"strconv"
)

// If a type has this function, its used instead of using reflection to determine a values path nodes.
type HasPathNodes interface {
	PathNodes() []string
}

// Inspects the given value for path nodes.
// A path node could be map keys, slice indices, or struct fields.
// If the value does not have any, nil is returned.
func GetPathNodes(v any) []string {
	c := Concrete(v)

	if hasNodes, ok := c.Interface().(HasPathNodes); ok {
		return hasNodes.PathNodes()
	}

	switch c.Kind() {
	case reflect.Map:
		mapKeys := c.MapKeys()
		nodes := make([]string, len(mapKeys))
		for i, mapKey := range mapKeys {
			nodes[i] = ToString(mapKey.Interface())
		}
		return nodes
	case reflect.Array, reflect.Slice:
		n := c.Len()
		nodes := make([]string, n)
		for i := 0; i < n; i++ {
			nodes[i] = strconv.Itoa(i)
		}
		return nodes
	case reflect.Struct:
		nodes := make([]string, 0, c.NumField())

		var traverseFields func(s reflect.Type)

		traverseFields = func(st reflect.Type) {
			for i := 0; i < st.NumField(); i++ {
				field := st.Field(i)
				if field.Anonymous {
					traverseFields(st.Field(i).Type)
				} else {
					nodes = append(nodes, field.Name)
				}
			}
		}

		traverseFields(c.Type())
		return nodes
	}

	return nil
}

// A path of keys/fields/indices to a value that can be gotten or set.
type Path []string

type HasGetPathNode interface {
	GetPathNode(node string) (any, error)
}

func (p Path) get(rv reflect.Value, node string) (reflect.Value, error) {
	c := Concrete(rv)

	if hasGet, ok := c.Interface().(HasGetPathNode); ok {
		v, err := hasGet.GetPathNode(node)
		return Reflect(v), err
	}

	switch c.Kind() {
	case reflect.Map:
		if c.IsNil() {
			return reflect.Value{}, fmt.Errorf("map is nil")
		}

		key, err := DecodeType(c.Type().Key(), node)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("error parsing key from node %s: %w", node, err)
		}

		return c.MapIndex(Reflect(key)), nil
	case reflect.Slice:
		if c.IsNil() {
			return reflect.Value{}, fmt.Errorf("slice is nil")
		}

		i, err := strconv.Atoi(node)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("error parsing array index from node %s: %w", node, err)
		}

		el := c.Index(i)
		// if el.CanAddr() {
		// 	el = el.Addr()
		// }

		return el, nil
	case reflect.Array:
		i, err := strconv.Atoi(node)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("error parsing slice index from node %s: %w", node, err)
		}

		el := c.Index(i)
		// if el.CanAddr() {
		// 	el = el.Addr()
		// }

		return el, nil
	case reflect.Struct:
		field := c.FieldByName(node)
		if !field.IsValid() {
			return field, fmt.Errorf("invalid field '%s' on type '%v'", node, c.Type())
		}

		return field, nil
	}

	return c, fmt.Errorf("unsupported path type '%v'", c.Type())
}

// Gets the value at this path for the given v.
func (p Path) Get(v any) (reflect.Value, error) {
	var err error
	rv := Reflect(v)
	for _, node := range p {
		rv, err = p.get(rv, node)
		if !rv.IsValid() || err != nil {
			return rv, err
		}
	}
	return rv, nil
}

// Sets the value at this path for the given v.
func (p Path) Set(v any, value any) error {
	rv, err := p.Get(v)
	if !rv.IsValid() || err != nil {
		return err
	}
	// fmt.Printf("set %v to %v\n", rv.String(), value)
	rv.Set(Reflect(value))
	return nil
}

// Sets the string at this path for the given v.
func (p Path) SetString(v any, s string) error {
	rv, err := p.Get(v)
	if !rv.IsValid() || err != nil {
		return fmt.Errorf("invalid path %v for type %v: %w", p, Reflect(v).Type(), err)
	}
	return Decode(rv, s)
}

// Sets the string at this path for the given v.
func (p Path) Next(node string) Path {
	newPath := append(Path{}, p...)
	newPath = append(newPath, node)
	return newPath
}

// A reference to a value in a path
type Ref struct {
	val  reflect.Value
	path Path
}

// A new reference given a value. If the set methods will be called
// this needs to be a pointer.
func NewRef(v any) Ref {
	return Ref{
		val:  Reflect(v),
		path: Path{},
	}
}

// Returns the available nodes based on the value in this reference.
func (r Ref) Nodes() ([]string, error) {
	val, err := r.Get()
	if err != nil {
		return nil, err
	}
	return GetPathNodes(val), nil
}

// Returns the current path for this reference.
func (r Ref) Path() []string {
	return r.path[:]
}

// Returns a reference to an inner value to the referenced value.
func (r Ref) Next(node string) Ref {
	return Ref{
		val:  r.val,
		path: r.path.Next(node),
	}
}

// Gets the referenced value.
func (r Ref) Get() (reflect.Value, error) {
	return r.path.Get(r.val)
}

// Sets the referenced value.
func (r Ref) Set(value any) error {
	return r.path.Set(r.val, value)
}

// Sets the referenced value from a string.
func (r Ref) SetString(value string) error {
	return r.path.SetString(r.val, value)
}
