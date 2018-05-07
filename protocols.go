package skink

// Node is the generic interface implemented by every node in a configuration
// tree in Skink.
type Node interface {
	// Name gets the case-insensitive name of the Node
	Name() String

	// Parent gets the node's Parent.  A nil Parent indicates a root node.
	Parent() Node

	// Class gets the node's Class in order to determine dynamic/runtime
	// properties of the Node.
	Class() Class

	// Children gets the Node's direct child nodes in a NodeMap
	Children() NodeMap
}

// InitNoder is implemented by any node that requires initialization after its
// child Nodes have been initialized.  Sibling nodes are all initialized
// together but only once they all finish, does the current node get
// initialized.
type InitNoder interface {
	InitNode(sk *Skink) error
}

// StartNoder is implemented by any Node that starts executing when Skink is
// started.  All nodes are started concurrently.
type StartNoder interface {
	StartNode(sk *Skink, root Node) error
}

// Value is a special type of node that can represent itself as a Go value.
type Value interface {
	Node
	Value() interface{}
}

// A Class describes a Node type hierarcy.
type Class interface {
	// Name gets the name of the Class
	Name() String

	// Base gets the direct base class of this Class.  The NodeClass's Base
	// is nil.
	Base() Class

	// Alloc is used by NewNode to allocate a new Node instance.
	Alloc(nodedef *NodeDef) (Node, error)

	// Init initializes a Node based on a NodeDef.
	Init(node, parent Node, nodedef *NodeDef) error
}
