# Skink

## Development

### Classes

Skink's Class interface expands upon Go's simple type system to define classes
which can inherit from bases to implement new functionality.

#### Class.Init vs. Node.InitNode

Contrary to Go's simple type system, Skink has a Class interface that defines
a simple type inheritance system.  A Skink Class can inherit from another
Class.

Classes define an Alloc function.  The point of this function is only to ensure
that accessing the nodes' fields from the Class's Init and the Node's InitNode
(if it has an InitNode function) won't result in nil reference panics from
the Go runtime.  The Alloc function is only called on the Class that is
actually being instantiated.

Class's Init functions are called only on the class that is actually being
instantiated.  When initializing base classes' data, the Init function is
responsible for initializing the base classes itself.  Skink Classes have a
Base method to get the current Class's base class.  Initializing the bases
can be as simple as

```
if err := cls.Base().Init(self, parent, nodeDef); err != nil {
	return err
}
```
But usually the base class is implemented in Go as a field such as the
`skink.BasicNode` struct, so the correct invocation of the base Class's Init
function is:
```
if err := cls.Base().Init(&self.BasicNode, parent, nodeDef); err != nil {
	return err
}
```

