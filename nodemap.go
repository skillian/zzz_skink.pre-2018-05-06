package skink

import (
	"fmt"
	"strings"

	"github.com/skillian/errors"
)

// NodeMap describes an ordered collection of Nodes that can be retrieved from
// by Node name
type NodeMap interface {
	// AddNode adds a node into the NodeMap.  If overwrite is true, if a Node
	// with the same name already exists, it is overwritten.  If overwrite is
	// false but a node with the same name exists, an error is returned.
	AddNode(node Node, overwrite bool) error

	// Contains checks if the NodeMap contains the given Node.
	Contains(node Node) bool

	// GetName gets a Node by its name.  If found, it is returned, otherwise,
	// the returned Node is nil and an error is returned.
	GetName(name String) (Node, error)

	// GetIndex gets a Node by its index within the ordered NodeMap.  If the
	// sequence is in bounds, the Node is returned.  Otherwise, an error is
	// returned.
	GetIndex(index int) (Node, error)

	// Len gets the number of Nodes within the NodeMap
	Len() int

	// Nodes returns the NodeMap's contents as a slice of Nodes.
	Nodes() []Node

	// RemoveName removes a node by name.  If the node does not exist directly
	// within the NodeMap, an error is returned.
	RemoveName(name String) error

	// RemoveIndex removes a Node at an index within the ordered NodeMap.
	// If a node does not exist at the specified index, an error is returned.
	RemoveIndex(index int) error

	// Remove removes the given Node from the NodeMap.  If the node is not
	// present in the map, an error is returned.
	Remove(node Node) error
}

// nodemap is the reference implementation of the NodeMap interface.
type nodemap struct {
	index map[string]int
	pairs []namenode
}

type namenode struct {
	name string
	node Node
}

// NewNodeMap creates a new NodeMap with the given capacity.  If capacity is
// 0, a NodeMap is instantiated but nested objects are nil to reduce allocations.
// if capacity is < 0, a NodeMap is instantiated with some default initial
// capacity.  If capacity is > 0, the NodeMap is initialized with a capacity
// of that value.
func NewNodeMap(capacity int) NodeMap {
	return &nodemap{
		index: makeNodeMapIndex(capacity),
		pairs: makeNodeMapPairs(capacity),
	}
}

func (m *nodemap) AddNode(node Node, overwrite bool) error {
	m.init()
	key := node.Name().lower
	pair, ok := m.getnn(key)
	if ok && (!overwrite) {
		return errors.Errorf("node with name %v already exists", node.Name())
	}
	if !ok {
		pair = m.newnn(key)
	}
	pair.name = node.Name().lower
	pair.node = node
	return nil
}

func (m *nodemap) Contains(node Node) bool {
	pair, ok := m.getnn(node.Name().lower)
	return ok && pair.node == node
}

func (m *nodemap) GetName(name String) (Node, error) {
	key := name.lower
	pair, ok := m.getnn(key)
	if !ok {
		return nil, NodeNotFound{nil, name}
	}
	return pair.node, nil
}

func (m *nodemap) GetIndex(index int) (Node, error) {
	index, ok := GetTrueIndex(m.Len(), index)
	if !ok {
		return nil, IndexError{index, m.Len()}
	}
	return m.pairs[index].node, nil
}

// String represents the NodeMap as a string.
func (m *nodemap) String() string {
	strs := make([]string, m.Len())
	for i, pair := range m.pairs {
		strs[i] = fmt.Sprintf("%#v: %#v", pair.name, pair.node)
	}
	return fmt.Sprintf(
		"NodeMap{%s}",
		strings.Join(strs, ", "))
}

func (m *nodemap) Len() int {
	return len(m.pairs)
}

func (m *nodemap) Nodes() []Node {
	length := m.Len()
	nodes := make([]Node, length)
	written := m.NodesInto(nodes)
	if written != length {
		panic(fmt.Sprintf(
			"(*nodemap).NodesInto wrote %d but expected %d",
			written, length))
	}
	return nodes
}

// NodesInto writes this map's nodes into the provided slice.  The number of
// Nodes inserted into that list is returned in the written argument.  Check
// the length of the nodes slice vs. the length of the NodeMap to make sure you
// got all of the nodes!
func (m *nodemap) NodesInto(nodes []Node) (written int) {
	minimum := minint(len(nodes), m.Len())
	for i, pair := range m.pairs[:minimum] {
		nodes[i] = pair.node
	}
	return minimum
}

func (m *nodemap) RemoveName(name String) error {
	index, ok := m.index[name.lower]
	if !ok {
		return NodeNotFound{Name: name}
	}
	return m.RemoveIndex(index)
}

func (m *nodemap) RemoveIndex(index int) (err error) {
	index, ok := GetTrueIndex(m.Len(), index)
	if !ok {
		return IndexError{index, m.Len()}
	}
	pair := m.pairs[index]
	m.pairs = append(m.pairs[:index], m.pairs[index+1:]...)
	delete(m.index, pair.name)
	for _, pair := range m.pairs[index:] {
		m.index[pair.name]--
	}
	return nil
}

func (m *nodemap) Remove(node Node) (err error) {
	name := node.Name()
	pair, ok := m.getnn(name.lower)
	if !ok {
		return NodeNotFound{Name: name}
	}
	if pair.node != node {
		return errors.Errorf(
			"node with name %q exists (%v) but is different (from: %v)",
			name, pair.node, node)
	}
	return m.RemoveName(name)
}

// init ensures the map is initialized with non-nil values.
func (m *nodemap) init() {
	if m.index == nil {
		m.index = make(map[string]int, DefaultNodeMapCapacity)
		m.pairs = make([]namenode, 0, DefaultNodeMapCapacity)
	}
}

func (m *nodemap) newnn(key string) *namenode {
	index := m.Len()
	m.index[key] = index
	m.pairs = append(m.pairs, namenode{})
	return &m.pairs[index]
}

func (m *nodemap) getnn(key string) (*namenode, bool) {
	index, ok := m.index[key]
	if !ok {
		return nil, false
	}
	return &m.pairs[index], true
}

// DefaultNodeMapCapacity defines a default capacity of NodeMaps in an attempt
// to minimize constant allocations during Skink's startup.  If it turns out to
// be an unnecessary micro-optimization, feel free to remove it.
const DefaultNodeMapCapacity = 8

func makeNodeMapIndex(capacity int) map[string]int {
	if capacity == 0 {
		return nil
	}
	if capacity < 0 {
		return make(map[string]int, DefaultNodeMapCapacity)
	}
	return make(map[string]int)
}

func makeNodeMapPairs(capacity int) []namenode {
	if capacity == 0 {
		return nil
	}
	if capacity < 0 {
		return make([]namenode, 0, DefaultNodeMapCapacity)
	}
	return make([]namenode, 0, capacity)
}

// GetTrueIndex takes the length of the collection and a relative index where
// negative values represent indexes from the end of the collection. The
// "translated" index is returned as well as whether or not the index is
// within the collection's bounds.
func GetTrueIndex(length, index int) (out int, ok bool) {
	if index < 0 {
		index = length + index
	}
	return index, index >= 0 || index < length
}

func minint(a, b int) int {
	if a < b {
		return a
	}
	return b
}
