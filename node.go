package refstr

import (
	"reflect"
	"strconv"
)

// The get func for a node if supported
type NodeGet = func(n Node, rv reflect.Value) reflect.Value

// The set func for a node if supported
type NodeSet = func(n Node, rv reflect.Value, val reflect.Value) error

// A node is a part of a path that represents a struct field, map key,
// slice or array index, a function call, or a method on a struct.
// This is returned when inspecting the available nodes for a type or value.
type Node struct {
	KeyString string
	Key       any
	KeyType   reflect.Type
	Type      reflect.Type
	CopyOnly  bool
	Get       NodeGet
	Set       NodeSet
}

// Returns whether this node represents a dynamic node and not a concrete one.
// Maps and slices have dynamic nodes, these nodes don't have a Key or KeyString.
func (n Node) IsDynamic() bool {
	return n.Key == nil && n.KeyString == ""
}

// Returns if the node is ready only
func (n Node) IsReadOnly() bool {
	return n.Set == nil
}

// Returns if the node is write only
func (n Node) IsWriteOnly() bool {
	return n.Get == nil
}

// Returns a copy of this node for the given key. This is especially useful for
// dynamic nodes.
func (n Node) ForKey(key any) Node {
	if !n.IsDynamic() {
		return n
	}
	copy := n
	copy.Key = key
	copy.KeyString = ToString(key)
	return copy
}

// The available nodes for a type/value
type Nodes struct {
	InOrder []Node
	ByKey   map[string]Node
}

// Creates a new place to store nodes by order and key.
func NewNodes(initial []Node) *Nodes {
	nodes := &Nodes{
		InOrder: make([]Node, 0),
		ByKey:   make(map[string]Node),
	}
	if len(initial) > 0 {
		for _, i := range initial {
			nodes.Add(i)
		}
	}
	return nodes
}

// Adds the node to the nodes
func (nodes *Nodes) Add(n Node) {
	if _, exists := nodes.ByKey[n.KeyString]; !exists {
		nodes.InOrder = append(nodes.InOrder, n)
		nodes.ByKey[n.KeyString] = n
	}
}

// Returns the node for the given key
func (nodes Nodes) ForKey(key any) *Node {
	if len(nodes.InOrder) == 1 && nodes.InOrder[0].IsDynamic() {
		concrete := nodes.InOrder[0].ForKey(key)
		return &concrete
	} else {
		keyString := ToString(key)
		if node, exists := nodes.ByKey[keyString]; exists {
			return &node
		}
		return nil
	}
}

// Returns all availables key strings in this collection of nodes.
func (nodes Nodes) KeyStrings() []string {
	keys := make([]string, 0, len(nodes.InOrder))
	for _, n := range nodes.InOrder {
		if !n.IsDynamic() {
			keys = append(keys, n.KeyString)
		}
	}
	return keys
}

// Adds the node to the nodes
func (nodes Nodes) Clone() *Nodes {
	return NewNodes(nodes.InOrder)
}

var typeNodes map[reflect.Type]*Nodes = make(map[reflect.Type]*Nodes)

// Sets the nodes available for the given value type.
func SetNodes[V any](nodes []Node) {
	typeNodes[TypeOf[V]()] = NewNodes(nodes)
}

// Sets the nodes available for the given type.
func SetTypeNodes(rt reflect.Type, nodes []Node) {
	typeNodes[rt] = NewNodes(nodes)
}

// Inspects a given type and returns the available nodes. Inspecting a value
// is more accurate since some types have dynamic values (slices & maps).
func GetTypeNodes(rt reflect.Type) *Nodes {
	if nodes, ok := typeNodes[rt]; ok {
		return nodes
	}

	c := ConcreteType(rt)

	if nodes, ok := typeNodes[c]; ok {
		return nodes
	}

	nodes := NewNodes(nil)
	typeNodes[c] = nodes

	switch c.Kind() {
	case reflect.Map:
		nodes.Add(Node{
			KeyType:  c.Key(),
			Type:     c.Elem(),
			CopyOnly: true,
			Get:      mapGet,
			Set:      mapSet,
		})
	case reflect.Slice:
		nodes.Add(Node{
			KeyType: indexType,
			Type:    c.Elem(),
			Get:     indexGet,
			Set:     indexSet,
		})
	case reflect.Array:
		len := c.Len()
		elementType := c.Elem()
		for i := 0; i < len; i++ {
			nodes.Add(Node{
				Key:       i,
				KeyType:   indexType,
				KeyString: strconv.Itoa(i),
				Type:      elementType,
				Get:       indexGet,
				Set:       indexSet,
			})
		}
	case reflect.Func:
		if IsGetter(c, nil) {
			returnNodes := GetTypeNodes(c.Out(0))
			for _, rn := range returnNodes.InOrder {
				node := rn
				get := node.Get
				node.Get = func(n Node, rv reflect.Value) reflect.Value {
					return get(n, rv.Call([]reflect.Value{})[0])
				}
				node.Set = nil
				nodes.Add(node)
			}
		}
	case reflect.Struct:
		fields := c.NumField()
		for i := 0; i < fields; i++ {
			field := c.Field(i)
			if field.Anonymous {
				embeddedNodes := GetTypeNodes(field.Type)
				for _, n := range embeddedNodes.InOrder {
					nodes.Add(n)
				}
			} else {
				nodes.Add(Node{
					Key:       field.Name,
					KeyType:   fieldType,
					KeyString: field.Name,
					Type:      field.Type,
					Get:       getFieldGet(i),
					Set:       getFieldSet(i),
				})
			}
		}
	}

	p := reflect.PointerTo(c)
	if p.NumMethod() != c.NumMethod() {
		pnodes := nodes.Clone()
		typeNodes[p] = pnodes

		addMethodNodes(p, pnodes)
	}

	addMethodNodes(c, nodes)

	if exactNodes, ok := typeNodes[rt]; ok {
		return exactNodes
	}

	return nodes
}

// Adds getter and setter nodes on the given type to the given nodes.
func addMethodNodes(t reflect.Type, nodes *Nodes) {
	methods := t.NumMethod()
	for i := 0; i < methods; i++ {
		method := t.Method(i)
		if IsGetter(method.Type, t) {
			nodes.Add(Node{
				Key:       method.Name,
				KeyType:   fieldType,
				KeyString: method.Name,
				Type:      method.Type.Out(0),
				Get:       getMethodGet(i),
			})
		} else if IsSetter(method.Type, t) {
			nodes.Add(Node{
				Key:       method.Name,
				KeyType:   fieldType,
				KeyString: method.Name,
				Type:      method.Type.In(1),
				Set:       getMethodSet(i),
			})
		}
	}
}

// Inspects the given value for path nodes.
// A path node could be map keys, slice indices, or struct fields.
// If the value does not have any, nil is returned.
func GetValueNodes(v any) *Nodes {
	c := Concrete(v)
	p := PointerMaybe(v)

	switch c.Kind() {
	case reflect.Map:
		nodes := NewNodes(nil)

		mapKeys := c.MapKeys()
		mapKeyType := c.Type().Key()
		mapValue := c.Type().Elem()

		for _, mapKey := range mapKeys {
			key := mapKey.Interface()

			nodes.Add(Node{
				Key:       key,
				KeyString: ToString(key),
				KeyType:   mapKeyType,
				Type:      mapValue,
				CopyOnly:  true,
				Get:       mapGet,
				Set:       mapSet,
			})
		}

		addMethodNodes(c.Type(), nodes)

		return nodes
	case reflect.Slice:
		nodes := NewNodes(nil)

		n := c.Len()
		elemType := c.Type().Elem()

		for i := 0; i < n; i++ {
			nodes.Add(Node{
				Key:       i,
				KeyType:   indexType,
				KeyString: strconv.Itoa(i),
				Type:      elemType,
				Get:       indexGet,
				Set:       indexSet,
			})
		}

		addMethodNodes(c.Type(), nodes)

		return nodes
	default:
		return GetTypeNodes(p.Type())
	}
}
