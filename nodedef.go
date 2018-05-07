package skink

import (
	"net/url"
)

// NodeDef structs are used by Skink internally as a standard form that
// configuration files are translated into before being initialized.
type NodeDef struct {
	// Name of this node
	Name String

	// Parent references the direct parent of this node.
	Parent *NodeDef

	// ClassURI holds a URI of this node's class or type.
	ClassURI *url.URL

	// Children are the direct children of this node.
	Children []*NodeDef

	// Value holds a basic string of data
	Value string
}

var (
	// ValueString holds a skink.String with the value "Value".
	// "Value" is a special child name in Skink.
	ValueString = MakeString("Value")
)

// NewNodeDef creates a NodeDef.  The created NodeDef will reference its parent
// but it will not be added to its parent's Children (use parent.NewChild
// instead, for that).
func NewNodeDef(name String, parent *NodeDef, classuri *url.URL) *NodeDef {
	return &NodeDef{
		Name:     name,
		Parent:   parent,
		ClassURI: classuri,
		Children: make([]*NodeDef, 0, DefaultNodeMapCapacity),
	}
}

// NewChild creates a new nodedef and adds it to this node's children
func (n *NodeDef) NewChild(name String, classuri *url.URL) *NodeDef {
	child := NewNodeDef(name, n, classuri)
	n.Children = append(n.Children, child)
	//if name.Cmp(ValueString) == 0 {
	//	n.Value = child
	//}
	return child
}

// FindChild attempts to find a child node by its name.  This is an O(n) search
// so don't do it if you don't need it.  I only use it right now while getting
// unique names for child nodes.
func (n *NodeDef) FindChild(name String) *NodeDef {
	for _, child := range n.Children {
		if child.Name.Cmp(name) == 0 {
			return child
		}
	}
	return nil
}
