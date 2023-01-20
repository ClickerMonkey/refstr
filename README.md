# refstr

A go module for converting between strings and any value AND referencing a path from a value through its structure to an end value and getting and setting it.

**String Examples**:

```go
// A value already initialized
x := 0
e := refstr.Decode(&x, "45")

// Specify a type and string
v, e := refstr.DecodeType(refstr.TypeOf[bool](), "yes")

// A value not yet initialized
var y *int
e := refstr.Decode(&y, "56")

// A slice
var z []float32
e := refstr.Decode(&z, "[0.5 1 3.1415]")

// A map
var w map[string]int
e := refstr.Decode(&w, "map[A:1 B:2 C:3]")

// A struct
type Point { X, Y float32 }
var p Point
e := refstr.Decode(&p, "{X:4 Y:67}")

// Control how types are parsed further with your own decoder
dec := refstr.NewDecoder()
dec.Trues["yeppers"] = struct{}{}
var b bool
e := dec.Decode(&b, "yeppers")

```

With references you can follow a path of fields, maps, slice & array elements, and getter and setter functions to get and set a value. Once a set is done it can create all the elements in the path if they don't exist yet.

**Reference Examples**:

```go
type Name struct { First, Last string }
func (n Name) Full() string { return n.Last + ", " + n.First }
type Person struct { Name Name }
type Persons struct { ByName map[string]Person }

p := Persons{}
pref := refstr.NewRef(&p)
// references John's name in the map, but none of it exists yet.
johnName := pref.Nexts([]any{"ByName", "John", "Name"})
// creates john in the map and sets his first name
johnName.Next("First").Set("John") 
// sets his last name
johnName.Next("Last").Set("Doe")
// gets his full name from the method Full()
full, _ := johnName.Next("Full").Get()
```