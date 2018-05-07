package skink

import (
	"strings"
	"sync"

	"github.com/skillian/errors"
)

// LeafNode defines a node that cannot have children.  The Children method
// always returns nil.
type LeafNode struct {
	NodeClass  Class
	NodeName   String
	NodeParent Node
}

// Class gets the LeafNode's skink Class.
func (n *LeafNode) Class() Class {
	return n.NodeClass
}

// Name gets the name of the skink Node.
func (n *LeafNode) Name() String {
	return n.NodeName
}

// Parent gets the node's parent.
func (n *LeafNode) Parent() Node {
	return n.NodeParent
}

// Children gets a NodeMap of the Node's children.
func (n *LeafNode) Children() NodeMap {
	return nil
}

// BasicNode is a struct bundling together the minimum Node interface's
// requirements.
type BasicNode struct {
	LeafNode
	NodeChildren NodeMap
}

// NodePathSeparator is the string that separates components of a Node's path
// from one another to describe the hierarchy.
const NodePathSeparator = "."

// FindNodes returns a function that iterates over a root Node's descendants
// to find nodes that match the filter.  Every time that returned function
// is called, the next matching node is returned and the bool returned value
// is true.  When all descendants are traversed and there are no more matches,
// subsequent calls to this returned function will result in nil, false.
func FindNodes(root Node, filter func(n Node) bool) func() (Node, bool) {
	nodes := make([]Node, 1, DefaultNodeMapCapacity)
	nodes[0] = root
	return func() (Node, bool) {
		for {
			logger.Debug1("nodes: %#v", nodes)
			if len(nodes) == 0 {
				return nil, false
			}
			node := nodes[0]
			nodes = append(nodes[1:], node.Children().Nodes()...)
			if filter(node) {
				return node, true
			}
		}
	}
}

// FindNode finds a single node matching the given predicate
func FindNode(root Node, predicate func(n Node) bool) (Node, bool) {
	return FindNodes(root, predicate)()
}

// ConcurrentFunc is executed in calls to ForEach.
type ConcurrentFunc func(node Node) error

// ForEach executes a ConcurrentFunc on every node in the node iterator
// concurrently and waits for them to all finish before returning.  If any of
// the ConcurrentFuncs returns an error, that/those errors are returned in a
// ConcurrentErrors.
func ForEach(nodeIter func() (Node, bool), f ConcurrentFunc) *ConcurrentErrors {
	wg := sync.WaitGroup{}
	ce := NewConcurrentErrors()
	for {
		node, ok := nodeIter()
		if !ok {
			break
		}
		wg.Add(1)
		go func(n Node) {
			err := f(n)
			if err != nil {
				ce.Add(err)
			}
			wg.Done()
		}(node)
	}
	wg.Wait()
	if ce.Len() == 0 {
		return nil
	}
	return ce
}

// ForEachInSlice concurrently executes a function on each node in the given
// slice.
func ForEachInSlice(nodes []Node, f ConcurrentFunc) *ConcurrentErrors {
	i := 0
	iter := func() (Node, bool) {
		if i >= len(nodes) {
			return nil, false
		}
		i++
		return nodes[i-1], true
	}
	return ForEach(iter, f)
}

// FindParents starts at a child and goes through it's "ancestors" checking
// if any match a predicate.  When a match is found, it is returned, if not,
// nil, false is returned.
func FindParents(node Node, predicate func(n Node) bool) func() (Node, bool) {
	return func() (Node, bool) {
		for {
			if node == nil {
				return nil, false
			}
			child := node
			node = node.Parent()
			if predicate(child) {
				return child, true
			}
		}
	}
}

// GetChildByPath traverses a path from a parent node to a child and gets that
// child node.
func GetChildByPath(node Node, path string) (child Node, err error) {
	parts := strings.Split(path, NodePathSeparator)
	for _, part := range parts {
		name := MakeString(part)
		child, err = node.Children().GetName(name)
		if err != nil {
			return nil, NodeNotFound{Parent: node, Name: name}
		}
		node = child
	}
	return node, nil
}

// GetPath gets the full path to the given node as a string
func GetPath(node Node) string {
	parents := make([]Node, 1, DefaultNodeMapCapacity)
	parents[0] = node
	iter := FindParents(node, TruePred)
	for parent, ok := iter(); ok; parent, ok = iter() {
		parents = append(parents, parent)
	}
	reversed := make([]string, len(parents))
	for i, parent := range parents {
		reversed[len(parents)-1-i] = parent.Name().String()
	}
	return strings.Join(reversed, NodePathSeparator)
}

// NewNode constructs an instance of the given class with the given parent.
func NewNode(cls Class, parent Node, nodeDef *NodeDef) (Node, error) {
	node, err := cls.Alloc(nodeDef)
	if err != nil {
		return nil, err
	}
	if err = cls.Init(node, parent, nodeDef); err != nil {
		return nil, err
	}
	return node, nil
}

// TruePred is a node predicate function that always returns true.
func TruePred(node Node) bool {
	return true
}

func allocBasicNode(nodeDef *NodeDef) (Node, error) {
	return new(BasicNode), nil
}

// InitLeafNode initializes a LeafNode
func InitLeafNode(leaf, parent Node, nodeDef *NodeDef) (err error) {
	n, ok := leaf.(*LeafNode)
	if !ok {
		return errors.Errorf(
			"InitLeafNode cannot initialize %v (type: %T)", leaf, leaf)
	}
	n.NodeClass, err = GetClassByURI(nodeDef.ClassURI)
	if err != nil {
		return errors.ErrorfWithCause(err, "failed to initialize Node: %v", err)
	}
	n.NodeName = nodeDef.Name
	n.NodeParent = parent
	return nil
}

func initBasicNode(self, parent Node, nodeDef *NodeDef) (err error) {
	n, ok := self.(*BasicNode)
	if !ok {
		return errors.Errorf("initNode cannot initialize a non-*node (got %v)", self)
	}
	if err = InitLeafNode(&n.LeafNode, parent, nodeDef); err != nil {
		return err
	}
	n.NodeChildren = NewNodeMap(len(nodeDef.Children))
	return nil
}

// Children gets the BasicNode's children.
func (n *BasicNode) Children() NodeMap {
	return n.NodeChildren
}
