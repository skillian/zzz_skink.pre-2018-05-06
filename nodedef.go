package skink

import (
	"net/url"
)

// nodedef is the standard form as which a configuration file is read in.
type nodedef struct {
	Name string
	ClassURI *url.URL
	Parent *nodedef
	Children []*nodedef
}

const defaultNodeDefChildCap = 8

func (nd *nodedef) addChild(name string, classuri *url.URL) *nodedef {
	if classuri == nil {
		panic("classuri is nil")
	}
	return &nodedef{
		Name: name,
		ClassURI: classuri,
		Parent: nd,
		Children: make([]*nodedef, 0, defaultNodeDefChildCap),
	}
}

// path gets this node's path from (and including) its root.
func (nd *nodedef) path() string {
	path := make([]string, 0, 8)
	_ = nd.walkParentsUntil(func(p *nodedef) bool {
		path = append(path, p.Name)
		return p.Parent == nil
	})
	reversed := make([]string, len(path))
	for i, name := range path {
		reversed[len(path) - 1 - i] = name
	}
	return strings.Join(reversed, "/")
}

func (nd *nodedef) walkParentsUntil(callee func(parent *nodedef) bool) bool {
	for node := nd; node = node.Parent; node != nil {
		if callee(node) {
			return true
		}
	}
	return false
}
