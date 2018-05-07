package skink

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/skillian/errors"
)

// LoadXMLFile loads an XML file from the given URI path into a collection of
// NodeDefs.
func LoadXMLFile(uri *url.URL) (nodedef *NodeDef, err error) {
	if !CanLoadXMLFile(uri) {
		return nil, errors.Errorf("cannot load URI %v", uri)
	}
	file, err := os.Open(GetURIPath(uri))
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to open file %v for reading: %v",
			uri.Path, err)
	}
	defer CatchDeferred(&err, file.Close)
	nodedef, err = newXMLFileLoader(file).Load()
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to load URI %v: %v",
			uri, err)
	}
	return nodedef, err
}

// CanLoadXMLFile checks if LoadXMLFile can load from the given URI.  This is
// just based on the data within the URI itself and doesn't actually ensure
// that loading will work.
func CanLoadXMLFile(uri *url.URL) bool {
	logger.Debug2("CanLoadXMLFile of URI %v (%#v)", uri, uri)
	if uri.Scheme != "file" {
		logger.Debug1("URI %v scheme is not \"file\"", uri)
		return false
	}
	if strings.ToLower(path.Ext(GetURIPath(uri))) != ".xml" {
		logger.Debug1("URI %v path extension is not \".xml\"", uri)
		return false
	}
	return true
}

type xmlFileLoader struct {
	decoder  *xml.Decoder
	elements []xml.StartElement
	nodedefs []*NodeDef
	rootdef  *NodeDef
}

func newXMLFileLoader(r io.Reader) *xmlFileLoader {
	return &xmlFileLoader{
		decoder:  xml.NewDecoder(r),
		elements: make([]xml.StartElement, 0, 8),
		nodedefs: make([]*NodeDef, 0, 8),
		rootdef:  nil,
	}
}

func (loader *xmlFileLoader) Load() (*NodeDef, error) {
	for {
		token, err := loader.decoder.Token()
		logger.Debug2("token: %#v, err: %#v", token, err)
		if err != nil {
			if err == io.EOF {
				return loader.rootdef, nil
			}
			return nil, err
		}
		switch e := token.(type) {

		case xml.StartElement:
			loader.elements = append(loader.elements, e)
			nodedef, err := loader.createNodeDef(e)
			if err != nil {
				return nil, err
			}
			if len(loader.nodedefs) == 0 {
				loader.rootdef = nodedef
			}
			loader.nodedefs = append(loader.nodedefs, nodedef)

		case xml.EndElement:
			loader.elements = loader.elements[:len(loader.elements)-1]
			loader.nodedefs = loader.nodedefs[:len(loader.nodedefs)-1]

		case xml.CharData:
			parent := loader.getParentNodeDef()
			if parent == nil {
				return nil, errors.Errorf(
					"CDATA cannot be the root node in a Skink configuration.")
			}
			parent.Value += string([]byte(e))
		}
	}
}

func (loader *xmlFileLoader) createNodeDef(e xml.StartElement) (nodedef *NodeDef, err error) {
	parent := loader.getParentNodeDef()
	name := loader.createNodeName(parent, e)
	uristring := createURIStringFromXMLName(e.Name)
	classuri, err := url.Parse(uristring)
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err,
			"failed to parse URI %v: %v",
			uristring, err)
	}
	if parent == nil {
		nodedef = NewNodeDef(name, parent, classuri)
	} else {
		nodedef = parent.NewChild(name, classuri)
	}
	for _, attr := range e.Attr {
		if attr.Name.Space == "xmlns" {
			// This is a namespace definition.  Ignore it.
			continue
		}
		_, err = loader.createAttrNodeDef(nodedef, attr)
		if err != nil {
			return nil, errors.ErrorfWithCause(
				err,
				"failed to create NodeDef from attribute %v on element %v: %v",
				attr, e, err)
		}
	}
	return nodedef, nil
}

func (loader *xmlFileLoader) createAttrNodeDef(parent *NodeDef, a xml.Attr) (nodeDef *NodeDef, err error) {
	classuri, err := getXMLAttrClassURI(a)
	if err != nil {
		return nil, err
	}
	child := parent.NewChild(MakeString(a.Name.Local), classuri)
	child.Value = a.Value
	return child, nil
}

func getXMLAttrClassURI(a xml.Attr) (*url.URL, error) {
	if a.Name.Space == "" {
		return StringClassURI, nil
	}
	return createURIFromXMLName(a.Name)
}

func createURIFromXMLName(name xml.Name) (uri *url.URL, err error) {
	uri, err = url.Parse(createURIStringFromXMLName(name))
	if err != nil {
		return nil, errors.ErrorfWithCause(
			err, "failed to create NodeDef from attribute: %v", err)
	}
	return uri, nil
}

func createURIStringFromXMLName(name xml.Name) string {
	if name.Space == "" {
		name.Space = "dynamic"
	}
	return strings.Join([]string{name.Space, "#", name.Local}, "")
}

func (loader *xmlFileLoader) getParentNodeDef() *NodeDef {
	length := len(loader.nodedefs)
	if length == 0 {
		return nil
	}
	return loader.nodedefs[length-1]
}

func (loader *xmlFileLoader) createNodeName(parent *NodeDef, e xml.StartElement) String {
	name := MakeString(getSuggestedXMLName(e))
	if parent != nil {
		numbered := name
		number := 2
		for {
			// Assuming whatever architecture we're on will roll over (maybe
			// 2's complement or 1's complement, etc.)
			if number < 2 {
				panic(
					fmt.Sprintf(
						"createNodeName rolled over while attempting to "+
							"create node %v under parent %v", e, parent))
			}
			if nodedef := parent.FindChild(numbered); nodedef == nil {
				return numbered
			}
			numbered = MakeString(fmt.Sprintf("%s%d", name, number))
			number++
		}
	}
	return name
}

var nameAttrString = MakeString("name")

func getSuggestedXMLName(e xml.StartElement) string {
	for _, attr := range e.Attr {
		attrName := MakeString(attr.Name.Local)
		if attr.Name.Space == "" && nameAttrString.Cmp(attrName) == 0 {
			return attr.Value
		}
	}
	return e.Name.Local
}

// todo(sk): Make this possible:
//
// <root>
//   <name>Root's name</name>
// </root>
//
// Right now, you can't do this.  The name node definition must be the
// element's tag name or an attribute in the node.
