package refstr

import (
	"errors"
	"reflect"
)

// Set might not be supported because a pointer was not provided or a node
// in the path is read-only
var ErrSetNotSupported = errors.New("set not available on this value")

// Get might not be supported because the node might not exist on the value
// or the last node in the path is write-only.
var ErrGetNotSupported = errors.New("get not available on this value")

// The value returned by get is not a valid value.
var ErrGetInvalid = errors.New("get has invalid result")

// A path of keys/fields/indices to a value that can be gotten or set.
type Path struct {
	root  reflect.Type
	nodes []Node
}

// Creates a new empty path of nodes.
func NewPath(root reflect.Type) Path {
	return Path{
		root:  root,
		nodes: make([]Node, 0),
	}
}

// Returns whether this path is empty.
func (p Path) IsEmpty() bool {
	return len(p.nodes) == 0
}

// Returns the key string representation of this path.
func (p Path) KeyStrings() []string {
	keys := make([]string, len(p.nodes))
	for i, n := range p.nodes {
		keys[i] = n.KeyString
	}
	return keys
}

// Returns the keys in this path.
func (p Path) Keys() []any {
	keys := make([]any, len(p.nodes))
	for i, n := range p.nodes {
		keys[i] = n.Key
	}
	return keys
}

// Returns the expected return type of this path.
func (p Path) Type() reflect.Type {
	if p.IsEmpty() {
		return p.root
	} else {
		return p.End().Type
	}
}

// Returns the next available nodes at the end of this path.
func (p Path) NextNodes() *Nodes {
	return GetTypeNodes(p.Type())
}

// Returns a new path following the next key, or returns nil if the
// given key is not valid.
func (p Path) Next(key any) *Path {
	nextNodes := p.NextNodes()
	nextNode := nextNodes.ForKey(key)

	if nextNode == nil {
		return nil
	}

	next := NewPath(p.root)
	next.nodes = append(next.nodes, p.nodes...)
	next.nodes = append(next.nodes, *nextNode)
	return &next
}

// Returns the root type for this path.
func (p Path) RootType() reflect.Type {
	return p.root
}

// Returns the nodes of this path.
func (p Path) Nodes() []Node {
	return p.nodes
}

// Returns the last node in the path or nil if the path is empty.
func (p Path) End() *Node {
	last := len(p.nodes) - 1
	if last == -1 {
		return nil
	} else {
		return &p.nodes[last]
	}
}

// Gets the value at this path for the given v.
func (p Path) Get(root any) (reflect.Value, error) {
	rv := Reflect(root)

	if p.IsEmpty() {
		return rv, nil
	}

	for _, node := range p.nodes {
		if node.Get == nil {
			return invalidValue, ErrGetNotSupported
		}
		rv = node.Get(node, rv)
		if !rv.IsValid() {
			return rv, ErrGetInvalid
		}
	}

	return rv, nil
}

// Sets the value at this path for the given v.
func (p Path) Set(root any, val any) error {
	rv := Reflect(root)

	last := len(p.nodes) - 1

	if last == -1 {
		if rv.CanSet() {
			Init(rv)
			rv.Set(Reflect(val))
			return nil
		}
		return ErrSetNotSupported
	}

	for i := 0; i <= last; i++ {
		node := p.nodes[i]
		if node.Set == nil {
			return ErrSetNotSupported
		}
		if node.Get == nil && i < last {
			return ErrGetNotSupported
		}
	}

	setBackTo := -1
	values := make([]reflect.Value, last+1)
	values[0] = rv

	for i := 0; i < last; i++ {
		node := p.nodes[i]

		if node.CopyOnly && setBackTo == -1 {
			setBackTo = i
		}

		next := node.Get(node, values[i])
		if !next.IsValid() {
			next = InitType(node.Type)
			if !next.IsValid() {
				return ErrGetInvalid
			}
			if setBackTo == -1 {
				setBackTo = i
			}
		}
		if !next.CanSet() {
			ptr := reflect.New(next.Type())
			ptr.Elem().Set(next)
			next = ptr.Elem()
		}
		if !InitValue(next, node.Type) {
			return ErrGetInvalid
		}
		values[i+1] = next
	}

	err := p.nodes[last].Set(p.nodes[last], values[last], Reflect(val))
	if err != nil {
		return err
	}

	if setBackTo != -1 {
		for k := last; k > setBackTo; k-- {
			prev := p.nodes[k-1]
			err = prev.Set(prev, values[k-1], values[k])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Sets the string at this path for the given v.
func (p Path) SetString(root any, s string) error {
	parsed, err := DecodeType(p.Type(), s)
	if err != nil {
		return err
	}
	return p.Set(root, parsed)
}

// A reference to a value in a path
type Ref struct {
	root reflect.Value
	path Path
}

// A new reference given a value. If the set methods will be called
// this needs to be a pointer.
func NewRef(v any) Ref {
	rv := Reflect(v)
	return Ref{
		root: rv,
		path: NewPath(rv.Type()),
	}
}

// Returns the reference path.
func (r Ref) Path() Path {
	return r.path
}

// Returns the available nodes based on the value in this reference.
func (r Ref) NextNodes() *Nodes {
	rv, err := r.Get()
	if err == nil {
		return nil
	}
	return GetValueNodes(rv)
}

// Returns a reference to an inner value to the referenced key.
func (r Ref) Next(key any) *Ref {
	nextPath := r.path.Next(key)
	if nextPath == nil {
		return nil
	}

	return &Ref{
		root: r.root,
		path: *nextPath,
	}
}

// Returns a reference to an inner value to the referenced keys.
func (r Ref) Nexts(keys []any) *Ref {
	if len(keys) == 0 {
		return &r
	} else if len(keys) == 1 {
		return r.Next(keys[0])
	} else {
		return r.Next(keys[0]).Nexts(keys[1:])
	}
}

// Gets the referenced value.
func (r Ref) Get() (reflect.Value, error) {
	return r.path.Get(r.root)
}

// Sets the referenced value.
func (r Ref) Set(value any) error {
	return r.path.Set(r.root, value)
}

// Sets the referenced value from a string.
func (r Ref) SetString(value string) error {
	return r.path.SetString(r.root, value)
}

var fieldType = TypeOf[string]()
var indexType = TypeOf[int]()
var errorType = TypeOf[error]()

var indexGet NodeGet = func(n Node, rv reflect.Value) reflect.Value {
	return Concrete(rv).Index(n.Key.(int))
}

var indexSet NodeSet = func(n Node, rv, val reflect.Value) error {
	index := n.Key.(int)
	c := Concrete(rv)
	if index < 0 {
		return ErrSetNotSupported
	}
	for index >= c.Len() && c.CanSet() {
		c.Set(reflect.Append(c, reflect.New(c.Type().Elem()).Elem()))
	}
	el := c.Index(index)
	if !el.CanSet() {
		return ErrSetNotSupported
	}
	el.Set(val)
	return nil
}

var mapGet NodeGet = func(n Node, rv reflect.Value) reflect.Value {
	return Concrete(rv).MapIndex(Reflect(n.Key))
}

var mapSet NodeSet = func(n Node, rv, val reflect.Value) error {
	Concrete(rv).SetMapIndex(Reflect(n.Key), val)
	return nil
}

var fieldGetMap map[int]NodeGet = make(map[int]NodeGet)

func getFieldGet(i int) NodeGet {
	fn := fieldGetMap[i]
	if fn == nil {
		fn = func(n Node, rv reflect.Value) reflect.Value {
			return Concrete(rv).Field(i)
		}
		fieldGetMap[i] = fn
	}
	return fn
}

var fieldSetMap map[int]NodeSet = make(map[int]NodeSet)

func getFieldSet(i int) NodeSet {
	fn := fieldSetMap[i]
	if fn == nil {
		fn = func(n Node, rv, val reflect.Value) error {
			f := Concrete(rv).Field(i)
			if !f.CanSet() {
				return ErrSetNotSupported
			}
			f.Set(val)
			return nil
		}
		fieldSetMap[i] = fn
	}
	return fn
}

var methodGetMap map[int]NodeGet = make(map[int]NodeGet)

func getMethodGet(i int) NodeGet {
	fn := methodGetMap[i]
	if fn == nil {
		fn = func(n Node, rv reflect.Value) reflect.Value {
			return rv.Method(i).Call([]reflect.Value{})[0]
		}
		methodGetMap[i] = fn
	}
	return fn
}

var methodSetMap map[int]NodeSet = make(map[int]NodeSet)

func getMethodSet(i int) NodeSet {
	fn := methodSetMap[i]
	if fn == nil {
		fn = func(n Node, rv, val reflect.Value) error {
			out := rv.Method(i).Call([]reflect.Value{val})
			if len(out) == 1 && out[0].Type().Implements(errorType) {
				return out[0].Interface().(error)
			}
			return nil
		}
		methodSetMap[i] = fn
	}
	return fn
}

var invalidValue = reflect.Value{}
