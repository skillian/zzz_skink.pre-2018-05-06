package skink

type Caller interface {
	Call(args NodeMap) (Node, error)
}

func Call(c Caller, args... Node) (Node, error) {
	
}