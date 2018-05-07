package skink

import (
	"net/url"
	"strings"

	"github.com/skillian/errors"
)

// String is a custom string type Skink uses to get custom comparison behavior
type String struct {
	value string
	lower string
}

// MakeString converts a Go string into a skink String.
func MakeString(value string) String {
	return String{value: value, lower: strings.ToLower(value)}
}

// Cmp performs a case-insensitive comparison of the two strings.
func (s String) Cmp(other String) int {
	return strings.Compare(s.lower, other.lower)
}

// Lower gets the string in an all-lower case form.
func (s String) Lower() string {
	return s.lower
}

// String implements fmt.Stringer
func (s String) String() string {
	return s.value
}

// StringNode is a skink String that implements the Node interface.
type StringNode struct {
	name   String
	parent Node
	String
}

// Name gets the name of the string in the configuration
func (s StringNode) Name() String { return s.name }

// Parent gets the string's parent Node
func (s StringNode) Parent() Node { return s.parent }

// Class gets the class of the string (StringClass)
func (s StringNode) Class() Class { return StringClass }

// Children returns a nil NodeMap (a string literal cannot have any child
// nodes).
func (s StringNode) Children() NodeMap { return nil }

// Value gets the string value as a Go string.
func (s StringNode) Value() interface{} { return s.value }

var (
	stringClassURIValue = url.URL{
		Scheme:   "import",
		Opaque:   "nodes",
		Fragment: "String",
	}

	// StringClass defines a basic String node
	StringClass = MustRegisterClassString(
		"import:nodes#String",
		stringClassType{})

	stringClassName = MakeString("String")

	// StringClassURI is a constant URI referring to the base StringNode
	// type
	StringClassURI = &stringClassURIValue
)

type stringClassType struct{}

func (c stringClassType) Name() String { return stringClassName }
func (c stringClassType) Base() Class  { return &nodeClassValue }

func (c stringClassType) Alloc(nodeDef *NodeDef) (Node, error) {
	return new(StringNode), nil
}

func (c stringClassType) Init(self, parent Node, nodeDef *NodeDef) error {
	if sn, ok := self.(*StringNode); ok {
		sn.name = nodeDef.Name
		sn.String = MakeString(nodeDef.Value)
	}
	return errors.Errorf("StringClass cannot init %T, only StringNode.", self)
}
