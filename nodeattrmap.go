package skink

import "github.com/skillian/errors"

// NodeTypeAttrMap defines an ordered collection of TypeAttrs used to get the
// value of a child node from a parent.
type NodeTypeAttrMap struct {
	index map[string]int
	pairs []TypeAttr
}

// NewNodeTypeAttrMap creates a new NodeTypeAttrMap with its inner attributes
// initialized
func NewNodeTypeAttrMap() *NodeTypeAttrMap {
	return &NodeTypeAttrMap{
		index: make(map[string]int),
		pairs: make([]TypeAttr, 0),
	}
}

// AddTypeAttr defines a new type attribute in the current NodeTypeAttrMap.
func (m *NodeTypeAttrMap) AddTypeAttr(a TypeAttr, overwrite bool) error {
	existing, exists := m.TypeAttrByName(a.Name)
	if exists && !overwrite {
		return errors.Errorf("Attribute %s already defined", a.Name)
	}
	if !exists {
		index := len(m.pairs)
		m.index[a.Name.Lower()] = index
		m.pairs = append(m.pairs, TypeAttr{})
		existing = &m.pairs[index]
	}
	*existing = a
	return nil
}

// Bind a Node to a NodeTypeAttrMap to get a NodeAttrMap.  This NodeAttrMap
func (m *NodeTypeAttrMap) Bind(node Node) NodeAttrMap {
	return NodeAttrMap{Node: node, NodeTypeAttrMap: m, dynamic: NewNodeMap(0)}
}

// ContainsKey returns whether or not a TypeAttr with the given key exists
func (m *NodeTypeAttrMap) ContainsKey(key string) bool {
	_, ok := m.index[key]
	return ok
}

// Len gets the length of the NodeTypeAttrMap
func (m *NodeTypeAttrMap) Len() int {
	return len(m.pairs)
}

// MustAddTypeAttr should be used in package-level var blocks to initialize
// a NodeTypeAttrMap
func (m *NodeTypeAttrMap) MustAddTypeAttr(a TypeAttr, overwrite bool) *NodeTypeAttrMap {
	PanicOnError(m.AddTypeAttr(a, overwrite))
	return m
}

// TypeAttrByKey gets a TypeAttr by its key in the NodeTypeAttrMap's index
func (m *NodeTypeAttrMap) TypeAttrByKey(key string) (*TypeAttr, bool) {
	index, ok := m.index[key]
	if !ok {
		return nil, false
	}
	return &m.pairs[index], true
}

// TypeAttrByName gets a TypeAttr from the NodeTypeAttrMap by its name.
func (m *NodeTypeAttrMap) TypeAttrByName(name String) (*TypeAttr, bool) {
	return m.TypeAttrByKey(name.Lower())
}

// TypeAttr defines a Node attribute and how to get that attribute's value.
type TypeAttr struct {
	Name   String
	Getter func(self Node) (Node, error)
	Setter func(self, value Node) error
}

// NodeAttrMap binds a NodeTypeAttrMap to a Node.  It also has a fallback NodeMap
// for dynamically defined attributes.
type NodeAttrMap struct {
	Node
	*NodeTypeAttrMap
	dynamic NodeMap
}

// AddNode adds a node to the dynamic NodeMap.
func (m NodeAttrMap) AddNode(node Node, overwrite bool) error {
	if a, ok := m.TypeAttrByName(node.Name()); ok {
		return a.Setter(m.Node, node)
	}
	return m.dynamic.AddNode(node, overwrite)
}

// Contains returns true if the given child node is contained in the node this
// NodeAttrMap is bound to.
func (m NodeAttrMap) Contains(node Node) bool {
	a, ok := m.NodeTypeAttrMap.TypeAttrByName(node.Name())
	if ok {
		if child, err := a.Getter(m.Node); err == nil {
			return child == node
		}
	}
	return m.dynamic.Contains(node)
}

// GetName tries to return a node from its NodeTypeAttrMap and falls back to its
// dynamic NodeMap
func (m NodeAttrMap) GetName(name String) (Node, error) {
	if a, ok := m.NodeTypeAttrMap.TypeAttrByName(name); ok {
		return a.Getter(m.Node)
	}
	return m.dynamic.GetName(name)
}

// GetIndex gets a child node by its index in the NodeAttrMap.  If the index
// is less than the length of its NodeTypeAttrMap, the attribute is retrieved from
// there.  If it's greater, subtract the length of the NodeTypeAttrMap from the
// index and get that index out of the dynamic NodeMap.
func (m NodeAttrMap) GetIndex(index int) (Node, error) {
	length := m.Len()
	index, ok := GetTrueIndex(length, index)
	if !ok {
		return nil, IndexError{Index: index, Length: length}
	}
	typelength := m.NodeTypeAttrMap.Len()
	if index < typelength {
		return m.NodeTypeAttrMap.pairs[index].Getter(m.Node)
	}
	return m.dynamic.GetIndex(index - typelength)
}

// Len gets the length of both the NodeTypeAttrMap and the dynamic NodeMap
func (m NodeAttrMap) Len() int {
	return m.NodeTypeAttrMap.Len() + m.dynamic.Len()
}

// Nodes gets all of the child nodes into a slice.
func (m NodeAttrMap) Nodes() []Node {
	nodes := make([]Node, m.Len())
	var err error
	for i, pair := range m.NodeTypeAttrMap.pairs {
		nodes[i], err = pair.Getter(m.Node)
		if err != nil {
			panic(err)
		}
	}
	_ = m.dynamic.(*nodemap).NodesInto(nodes[m.NodeTypeAttrMap.Len():])
	return nodes
}

// RemoveName removes a child node by its name in the attribute.  If the
// attribute is in the NodeTypeAttrMap, the removal will fail.
func (m NodeAttrMap) RemoveName(name String) error {
	if _, ok := m.TypeAttrByName(name); ok {
		return errors.Errorf("cannot remove type attribute %v", name)
	}
	return m.dynamic.RemoveName(name)
}

// RemoveIndex removes an attribute at the given index from the node.
// if the attribute is in the NodeTypeAttrMap, the removal will fail.
func (m NodeAttrMap) RemoveIndex(index int) error {
	mlen := m.Len()
	index, ok := GetTrueIndex(mlen, index)
	if ok {
		tamlen := m.NodeTypeAttrMap.Len()
		if index < tamlen {
			return errors.Errorf("cannot remove type attribute at index %d", index)
		}
		return m.dynamic.RemoveIndex(index - tamlen)
	}
	return IndexError{index, mlen}
}

// Remove will remove a node from the dynamic NodeMap.  If the node is present
// in the NodeTypeAttrMap, the removal will fail.
func (m NodeAttrMap) Remove(node Node) error {
	if _, ok := m.NodeTypeAttrMap.TypeAttrByName(node.Name()); ok {
		return errors.Errorf("cannot remove attribute from NodeTypeAttrMap")
	}
	return m.dynamic.Remove(node)
}
