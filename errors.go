package skink

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/skillian/errors"
)

// ClassNotFound is returned when an attempt is made to find a class but there
// is no such class registered
type ClassNotFound struct {
	*url.URL
}

// Error implements the error interface.
func (err ClassNotFound) Error() string {
	return fmt.Sprintf("Class %v not found", err.URL)
}

// NodeNotFound errors are returned when a requested node cannot be found.
type NodeNotFound struct {
	// Parent is the node under which another node was sought.  If the parent
	// is not known, this will be nil.
	Parent Node

	// Name stores the name of the node that was attempted to be retrieved.
	Name String
}

// MakeNodeNotFoundFromNode makes a NodeNotFound error from a Node instance.
func MakeNodeNotFoundFromNode(node Node) NodeNotFound {
	return NodeNotFound{Parent: nil, Name: node.Name()}
}

// MakeNodeNotFoundByName creates a NodeNotFound error from a parent and a
// name.
func MakeNodeNotFoundByName(parent Node, name String) NodeNotFound {
	return NodeNotFound{Parent: parent, Name: name}
}

// MakeNodeNotFoundByNameString makes a NodeNotFound error from a parent Node
// and a sought child node's name as a Go string.
func MakeNodeNotFoundByNameString(parent Node, name string) NodeNotFound {
	return MakeNodeNotFoundByName(parent, MakeString(name))
}

// Error implements the error interface.
func (n NodeNotFound) Error() string {
	extra := ""
	if n.Parent != nil {
		extra = "in parent " + GetPath(n.Parent)
	}
	return fmt.Sprintf("Node %s not found%s", n.Name, extra)
}

// IndexError is just like in Python, describing an index out of range.
type IndexError struct {
	Index  int
	Length int
}

// Error implements the error interface.
func (e IndexError) Error() string {
	return fmt.Sprintf(
		"cannot get index %d of collection with length %d",
		e.Index, e.Length)
}

// CatchDeferred ensures that an error returned by a deferred function is not
// discarded.  errptr should be a pointer to a named return value.
func CatchDeferred(errptr *error, errorers ...func() error) {
	if errptr == nil {
		panic("nil errptr parameter to CatchDeferred")
	}
	for _, errorer := range errorers {
		err := errorer()
		if err != nil {
			if *errptr != nil {
				*errptr = errors.Error{Err: err, Context: *errptr}
			} else {
				*errptr = err
			}
		}
	}
}

// ConcurrentErrors holds a collection of errors from concurrently executed
// functions.
type ConcurrentErrors struct {
	errors []error
}

// NewConcurrentErrors creates a new ConcurrentErrors slice.
func NewConcurrentErrors() *ConcurrentErrors {
	return new(ConcurrentErrors)
}

// Error concatenates the errors all together into a single error string
func (ce *ConcurrentErrors) Error() string {
	errors := make([]string, len(ce.errors)+1)
	errors[0] = fmt.Sprintf("%d errors occurred:", len(ce.errors))
	for i, err := range ce.errors {
		errors[i+1] = fmt.Sprintf("%3d:\t%s", i+1, err.Error())
	}
	return strings.Join(errors, "\n\t")
}

// Add an error to the collection of concurrent errors.
func (ce *ConcurrentErrors) Add(errs ...error) {
	ce.errors = append(ce.errors, errs...)
}

// Len gets the length of the ConcurrentErrors slice (that is, the number of
// bundled concurrent errors).
func (ce *ConcurrentErrors) Len() int {
	return len(ce.errors)
}

// PanicOnError is for initialization functions that should panic if their
// returned error value is not nil.
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
