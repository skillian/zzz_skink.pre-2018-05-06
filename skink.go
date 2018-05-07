package skink

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/skillian/errors"
	"github.com/skillian/logging"
)

// Skink is the context that holds and runs several Skink configurations.
type Skink struct {
	mutex    sync.RWMutex
	parent   *Skink
	children []*Skink

	roots []Node

	HTTPClient http.Client
	*logging.Logger
	Package string
	TempDir string

	uriloaders map[string][]*uriloader
}

// uriloader defines a function that can be called to convert the data in the
// given URI into a NodeDef tree that Skink can then use to create a Node
// tree.
type uriloader struct {
	loader  func(*url.URL) (*NodeDef, error)
	filter  func(*url.URL) bool
	schemes []string
}

var (
	// GlobalSkink is the default global Skink context from which manually
	// instantiated contexts should be created (or at least their root
	// parent should point at instead of being nil).
	GlobalSkink *Skink

	logger = logging.GetLogger("github.com/skillian/skink")
)

func init() {
	GlobalSkink = mustCreateSkink(nil, "github.com/skillian/skink")
}

// createSkink creates and initializes a new Skink context.
func createSkink(parent *Skink, pkg string) (*Skink, error) {
	tmppkgdir := path.Join(os.TempDir(), path.Dir(pkg))
	if err := os.MkdirAll(tmppkgdir, os.ModeDir); err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to create temporary directory root %v: %v",
			tmppkgdir, err)
	}
	tempdir, err := ioutil.TempDir(tmppkgdir, path.Base(pkg))
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to create temporary directory for new Skink context: %v",
			err)
	}
	sk := &Skink{
		mutex:      sync.RWMutex{},
		parent:     parent,
		children:   make([]*Skink, 0, 1),
		HTTPClient: http.Client{},
		Logger:     logging.GetLogger(pkg),
		Package:    pkg,
		TempDir:    tempdir,
		uriloaders: make(map[string][]*uriloader),
	}
	sk.RegisterURILoader(sk.loadhttp, nil, "http", "https")
	sk.RegisterURILoader(LoadXMLFile, CanLoadXMLFile, "file")
	return sk, nil
}

// mustCreateSkink panics if its underlying call to createSkink returns an
// error.
func mustCreateSkink(parent *Skink, pkg string) *Skink {
	sk, err := createSkink(parent, pkg)
	if err != nil {
		panic(err)
	}
	return sk
}

// CreateChild creates a new child skink context.  I'm not yet sure of a reason
// to do this yet.
func (sk *Skink) CreateChild(pkg string) (*Skink, error) {
	sk.mutex.Lock()
	defer sk.mutex.Unlock()
	child, err := createSkink(sk, pkg)
	if err != nil {
		return nil, err
	}
	sk.children = append(sk.children, child)
	return child, nil
}

// MustCreateChild creates a child Skink instance.  If an error occurs during
// creation, panic.
func (sk *Skink) MustCreateChild(pkg string) *Skink {
	child, err := sk.CreateChild(pkg)
	if err != nil {
		panic(err)
	}
	return child
}

// Parents creates a function that iterates up a Skink context's parents until
// a nil parent is reached.  The first call to the function returned by Parents
// yields the "self" Skink context.
func (sk *Skink) Parents() func() (*Skink, bool) {
	return func() (parent *Skink, ok bool) {
		if sk == nil {
			return nil, false
		}
		parent = sk
		sk = sk.parent
		return parent, true
	}
}

// RegisterURILoader registers a function that can load URIs for a provided
// list of URI schemes.  Multiple loaders can be defined for the same scheme.
func (sk *Skink) RegisterURILoader(loader func(*url.URL) (*NodeDef, error), filter func(*url.URL) bool, schemes ...string) {
	sk.mutex.Lock()
	defer sk.mutex.Unlock()
	ul := &uriloader{loader: loader, filter: filter, schemes: schemes}
	for _, scheme := range schemes {
		slice, ok := sk.uriloaders[scheme]
		if !ok {
			slice = make([]*uriloader, 0, 1)
		}
		slice = append(slice, ul)
		sk.uriloaders[scheme] = slice
	}
}

// CreateNodeFromURI creates a Node by loading from the given URI.
func (sk *Skink) CreateNodeFromURI(uri *url.URL) (Node, error) {
	nodedef, err := sk.CreateNodeDef(uri)
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to load URI %v: %v",
			uri, err)
	}
	return sk.CreateNode(nil, nodedef)
}

// CreateNodeDef creates a NodeDef tree from the configuration in the specified
// file.  That NodeDef is not initialized or converted to Nodes in any way
// by the createNodeDef function.
func (sk *Skink) CreateNodeDef(uri *url.URL) (*NodeDef, error) {
	schemes, ok := sk.getURILoadersForScheme(uri.Scheme)
	if !ok {
		return nil, errors.Errorf(
			"no URI loader registered for scheme: %s",
			uri.Scheme)
	}
	logger.Debug2("URI Loaders for scheme %v: %v", uri.Scheme, schemes)
	var lasterr error
	for i := range schemes {
		ul := schemes[len(schemes)-1-i]
		if ul.filter != nil && !ul.filter(uri) {
			continue
		}
		nodedef, err := ul.loader(uri)
		if err == nil {
			return nodedef, nil
		}
		lasterr = errors.ErrorfWithCauseAndContext(
			err,
			lasterr,
			"failed to load URI %v with loader %v: %v",
			uri, ul.loader, err)
	}
	if lasterr == nil {
		lasterr = errors.Errorf("no URI loader loaded %v", uri)
	}
	return nil, lasterr
}

// CreateNode creates a node under the given parent from the given NodeDef.
// CreateNode recursively creates the nodes under nodeDef, too.
func (sk *Skink) CreateNode(parent Node, nodeDef *NodeDef) (Node, error) {
	cls, err := GetClassByURI(nodeDef.ClassURI)
	if err != nil {
		if _, ok := err.(ClassNotFound); ok {
			cls, err = CreateDynamicClass(nodeDef.ClassURI)
			if err != nil {
				return nil, errors.ErrorfWithCause(
					err,
					"failed to create class dynamically: %v",
					err)
			}
		} else {
			return nil, errors.ErrorfWithCause(
				err,
				"failed to create node from NodeDef %v: %v",
				nodeDef, err)
		}
	}
	node, err := cls.Alloc(nodeDef)
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to allocate Node from Class %v: %v",
			cls.Name(), err)
	}
	err = cls.Init(node, parent, nodeDef)
	// cls.Init should have set the node's parent.
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to initialize Node %v from Class %v: %v",
			node, cls, err)
	}
	for _, childDef := range nodeDef.Children {
		child, err := sk.CreateNode(node, childDef)
		if err != nil {
			return nil, err
		}
		if err = node.Children().AddNode(child, false); err != nil {
			return nil, errors.ErrorfWithCause(
				err,
				"error adding child Node %v to parent Node %v: %v",
				child, parent, err)
		}
	}
	return node, nil
}

// InitNode initializes a node (after initializing all of if its child Nodes).
func (sk *Skink) InitNode(node Node) error {
	if node == nil {
		return nil
	}
	err := ForEachInSlice(node.Children().Nodes(), sk.InitNode)
	if err != nil {
		return errors.ErrorfWithCause(
			err,
			"failed to initialize node %v: %v",
			node, err)
	}
	if initnoder, ok := node.(InitNoder); ok {
		return initnoder.InitNode(sk)
	}
	return nil
}

// StartNode starts a node and all of its child Nodes.
func (sk *Skink) StartNode(root Node) error {
	nodes := FindNodes(root, TruePred)
	wg := sync.WaitGroup{}
	ce := NewConcurrentErrors()
	for {
		child, ok := nodes()
		if !ok {
			break
		}
		if startnoder, ok := child.(StartNoder); ok {
			wg.Add(1)
			go func(sn StartNoder) {
				logger.Debug1("Starting node %#v", sn)
				if err := sn.StartNode(sk, root); err != nil {
					ce.Add(err)
				}
				wg.Done()
			}(startnoder)
		}
	}
	wg.Wait()
	if ce.Len() == 0 {
		return nil
	}
	return ce
}

// StartURIStrings takes a collection of URI strings and starts their nodes.
func (sk *Skink) StartURIStrings(uris ...string) error {
	return errors.Errorf("StartURIStrings is not yet implemented")
}

func (sk *Skink) getURILoadersForScheme(scheme string) ([]*uriloader, bool) {
	sk.mutex.RLock()
	defer sk.mutex.RUnlock()
	schemes, ok := sk.uriloaders[scheme]
	return schemes, ok
}

// loadhttp downloads a file via HTTP to a temporary file and then tries to
// use (*Skink).createNodeDef to load that file.  This way, URI loaders only
// need to be able to load from the file URI scheme.
func (sk *Skink) loadhttp(uri *url.URL) (nodedef *NodeDef, err error) {
	path := path.Join(sk.TempDir, uri.Host, uri.Path)
	err = os.MkdirAll(path, os.ModeDir)
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to create temporary directory %q to download %v: %v",
			path, uri, err)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to open file %q for writing: %v",
			path, err)
	}
	defer CatchDeferred(&err, file.Close)
	resp, err := sk.HTTPClient.Get(uri.String())
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to get URI: %v: %v",
			uri, err)
	}
	defer CatchDeferred(&err, func() error {
		_, err := io.Copy(ioutil.Discard, resp.Body)
		return err
	}, resp.Body.Close)
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, err
	}
	return sk.CreateNodeDef(&url.URL{
		Scheme:   "file",
		Path:     path,
		Fragment: uri.Fragment,
	})
}

// GetURIPath gets the relative or full path in the URI.
func GetURIPath(uri *url.URL) string {
	if uri.Opaque == "" {
		return uri.Path
	}
	return uri.Opaque
}
