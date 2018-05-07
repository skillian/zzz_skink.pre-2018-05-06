package skink

import (
	"net/url"
	"strings"
	"sync"

	"github.com/skillian/errors"
)

var (
	nodeClassValue = nodeclass{
		name:        MakeString("Node"),
		base:        nil,
		allocator:   allocBasicNode,
		initializer: initBasicNode,
	}

	// NodeClass is the basic class that all other classes should directly or
	// indirectly reference as a base.
	NodeClass Class = &nodeClassValue

	classRegistryMutex = sync.RWMutex{}

	classRegistry map[string]Class
)

func init() {
	classRegistry = map[string]Class{
		"import:nodes#node": &nodeClassValue,
	}
}

// CreateDynamicClass creates a dynamic class from the given URI and registers
// that class.  The registered class's Alloc and Init functions are the same
// as the base class's.
func CreateDynamicClass(uri *url.URL) (Class, error) {
	cls, err := GetClassByURI(uri)
	if cls != nil {
		return nil, errors.Errorf("Class %s already registered", cls.Name())
	}
	logger.Debug1("Creating dynamic class for URI: %v", uri)
	base := GetBaseClassFromURI(uri)
	cls = &nodeclass{
		name:        MakeString(uri.Fragment),
		base:        base,
		allocator:   base.Alloc,
		initializer: base.Init,
	}
	err = RegisterClass(uri, cls)
	if err != nil {
		logger.Error2("failed to register dynamic class %v: %v", uri, err)
		return cls, err
	}
	return cls, nil
}

// GetBaseClassFromURI tries to get a base Node class from the given URL.  If
// one cannot be found, NodeClass is returned instead.
func GetBaseClassFromURI(uri *url.URL) Class {
	baseuri := &url.URL{
		Scheme:     uri.Scheme,
		Opaque:     uri.Opaque,
		User:       uri.User,
		Host:       uri.Host,
		Path:       uri.Path,
		RawPath:    uri.RawPath,
		ForceQuery: uri.ForceQuery,
		RawQuery:   uri.RawQuery,
		Fragment:   "Node",
	}
	basecls, err := GetClassByURI(baseuri)
	if err != nil {
		logger.Info2("failed to get registered base Class %v: %v", baseuri, err)
		return NodeClass
	}
	return basecls
}

// GetClassByURI gets a registered class by its URI. If the class is not found,
// an error is returned.
func GetClassByURI(uri *url.URL) (Class, error) {
	classRegistryMutex.RLock()
	defer classRegistryMutex.RUnlock()
	key := strings.ToLower(uri.String())
	cls, ok := classRegistry[key]
	if !ok {
		return nil, ClassNotFound{URL: uri}
	}
	return cls, nil
}

// RegisterClass registers a class by its URI in the global class registry.
func RegisterClass(uri *url.URL, cls Class) error {
	return RegisterClassString(uri.String(), cls)
}

// RegisterClassString registers a class with a URI that is already in string
// form.
func RegisterClassString(uri string, cls Class) error {
	classRegistryMutex.Lock()
	defer classRegistryMutex.Unlock()
	key := strings.ToLower(uri)
	existing, ok := classRegistry[key]
	if ok {
		return errors.Errorf("Class %v is already registered under URI %v", existing, uri)
	}
	classRegistry[key] = cls
	logger.Debug2("Registered class %v under URI %v", cls, uri)
	return nil
}

// MustRegisterClassString registers the given class and returns it so it can
// be assigned in a var block.
func MustRegisterClassString(uri string, cls Class) Class {
	if err := RegisterClassString(uri, cls); err != nil {
		panic(err)
	}
	return cls
}

type nodeclass struct {
	name        String
	base        Class
	allocator   func(nodeDef *NodeDef) (Node, error)
	initializer func(self, parent Node, nodeDef *NodeDef) error
}

func (cls *nodeclass) Name() String {
	return cls.name
}

func (cls *nodeclass) Base() Class {
	return cls.base
}

func (cls *nodeclass) Alloc(nodedef *NodeDef) (Node, error) {
	return cls.allocator(nodedef)
}

func (cls *nodeclass) Init(self, parent Node, nodedef *NodeDef) error {
	return cls.initializer(self, parent, nodedef)
}
