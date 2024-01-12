package gowhistler

import (
	"fmt"
	"github.com/beevik/etree"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type WSDL struct {
	UrlToNameSpaceMapping map[string]string
	TargetNamespace       string

	Messages []Message
	Ports    []Port
	Bindings []Binding
	Services []Service
	TypeMap  map[string]ElementType // full namespace : name -> element

	Elements []Element
	Types    []ElementType
}

func getPrefixToNamespaceMap(doc *etree.Document) map[string]string {
	prefixToNamespace := make(map[string]string)
	// get namespace mapping
	for _, attr := range doc.Root().Attr {
		if attr.Space == "xmlns" {
			prefixToNamespace[attr.Key] = attr.Value
		}
	}

	return prefixToNamespace
}

func Parse(url string) (*WSDL, error) {
	ret := &WSDL{
		UrlToNameSpaceMapping: make(map[string]string),
		TypeMap:               make(map[string]ElementType),
	}

	doc, err := getWSDL(url)
	if err != nil {
		return nil, err
	}

	prefixes := getPrefixToNamespaceMap(doc)
	// get namespace mapping
	for _, attr := range doc.Root().Attr {
		if attr.Space == "xmlns" {
			ret.UrlToNameSpaceMapping[attr.Value] = attr.Key
		}
	}
	ret.TargetNamespace = doc.Root().SelectAttrValue("targetNamespace", "")

	// parse types
	wsdlNamespaceKey, ok := ret.UrlToNameSpaceMapping["http://schemas.xmlsoap.org/wsdl/"]
	if !ok {
		return nil, errors.New("Could not find namespace for wsdl")
	}
	_ = wsdlNamespaceKey

	types := doc.FindElements(`//types[namespace-prefix()='` + wsdlNamespaceKey + `']/schema`)
	for _, tpelm := range types {
		elements, types, err := ParseSchema(tpelm, tpelm.SelectAttrValue("targetNamespace", ""), prefixes, url, 0)
		if err != nil {
			return nil, err
		}
		ret.Elements = append(ret.Elements, elements...)
		ret.Types = append(ret.Types, types...)
	}

	// parse messages
	xmlMessages := doc.FindElements(`//message[namespace-prefix()='` + wsdlNamespaceKey + `']`)
	for _, msgelm := range xmlMessages {
		msg, err := ParseMesssage(msgelm, prefixes)
		if err != nil {
			return nil, err
		}
		ret.Messages = append(ret.Messages, msg)
	}

	xmlPorts := doc.FindElements(`//portType[namespace-prefix()='` + wsdlNamespaceKey + `']`)
	for _, portelm := range xmlPorts {
		port, err := ParsePort(portelm)
		if err != nil {
			return nil, err
		}
		ret.Ports = append(ret.Ports, port)
	}

	xmlBindings := doc.FindElements(`//binding[namespace-prefix()='` + wsdlNamespaceKey + `']`)
	for _, bindingelm := range xmlBindings {
		binding, err := ParseBinding(bindingelm)
		if err != nil {
			return nil, err
		}
		ret.Bindings = append(ret.Bindings, binding)
	}

	xmlServices := doc.FindElements(`//service[namespace-prefix()='` + wsdlNamespaceKey + `']`)
	for _, serviceelm := range xmlServices {
		service, err := ParseService(serviceelm)
		if err != nil {
			return nil, err
		}
		ret.Services = append(ret.Services, service)
	}

	// build type map
	for _, tp := range ret.Types {
		if tp.NameSpace == "" {
			tp.NameSpace = ret.TargetNamespace
		}
		ret.TypeMap[tp.FullName()] = tp
	}

	for _, elm := range ret.Elements {
		if elm.NameSpace == "" {
			elm.NameSpace = ret.TargetNamespace
		}

		if !strings.Contains(elm.ElementType, ":") {
			elm.ElementType = ret.TargetNamespace + ":" + elm.ElementType
		}

		elm.ElementType = strings.TrimPrefix(elm.ElementType, "http://www.w3.org/2001/XMLSchema")

		switch elm.ElementType {
		case ":date", ":string", ":int", ":boolean", ":dateTime", ":base64Binary", ":NMTOKEN", ":NCName":
			continue
		}

		tp, ok := ret.TypeMap[elm.ElementType]
		if !ok {
			panic("Could not find type")
		}
		ret.TypeMap[elm.FullName()] = tp
	}

	return ret, nil

}

type MessagePart struct {
	Name    string
	Element string
}

type Message struct {
	Name  string
	Parts []MessagePart
}

type Port struct {
	Name       string
	Operations []PortOperation
}

type PortOperation struct {
	Name   string
	Input  PortOperationComponent
	Output PortOperationComponent
	Fault  PortOperationComponent
}

type PortOperationComponent struct {
	Message string
	Name    string
}

type Binding struct {
	Name       string
	Type       string
	Operations []BindingOperation
}

type BindingOperation struct {
	Name       string
	SoapAction string
	Input      []BindingOperationComponent
	Output     []BindingOperationComponent
	Fault      []BindingOperationComponent
}

type BindingOperationComponent struct {
	In      string
	Use     string
	Parts   []string
	Message string
	Name    string
}

type Service struct {
	Name  string
	Ports []ServicePort
}

type ServicePort struct {
	Name            string
	Binding         string
	AddressLocation string
}

func ParseService(elm *etree.Element) (service Service, err error) {
	ret := Service{
		Name: elm.SelectAttrValue("name", ""),
	}

	for _, child := range elm.ChildElements() {
		if child.Tag != "port" {
			continue
		}

		port := ServicePort{
			Name:    child.SelectAttrValue("name", ""),
			Binding: child.SelectAttrValue("binding", ""),
		}
		for _, child := range child.ChildElements() {
			if child.Tag == "address" {
				port.AddressLocation = child.SelectAttrValue("location", "")
			}
		}

		ret.Ports = append(ret.Ports, port)
	}
	return ret, nil
}

func ParseBinding(elm *etree.Element) (binding Binding, err error) {
	ret := Binding{
		Name: elm.SelectAttrValue("name", ""),
		Type: elm.SelectAttrValue("type", ""),
	}

	for _, child := range elm.ChildElements() {
		if child.Tag != "operation" {
			continue
		}

		op := BindingOperation{
			Name: child.SelectAttrValue("name", ""),
		}

		for _, child := range child.ChildElements() {
			switch child.Tag {
			case "operation":
				op.SoapAction = child.SelectAttrValue("soapAction", "")
			case "input":
				op.Input = ParseBindingOperationComponent(child)
			case "output":
				op.Output = ParseBindingOperationComponent(child)
			case "fault":
				op.Fault = ParseBindingOperationComponent(child)
			}
		}

		ret.Operations = append(ret.Operations, op)
	}

	return ret, nil
}

func ParseBindingOperationComponent(elm *etree.Element) []BindingOperationComponent {
	var ret []BindingOperationComponent

	for _, child := range elm.ChildElements() {
		component := BindingOperationComponent{
			In:      child.Tag,
			Use:     child.SelectAttrValue("use", ""),
			Message: child.SelectAttrValue("message", ""),
			Name:    child.SelectAttrValue("name", ""),
		}

		for _, attr := range child.Attr {
			if attr.Key == "part" || attr.Key == "parts" {
				component.Parts = append(component.Parts, attr.Value)
			}
		}

		ret = append(ret, component)

	}
	return ret
}

func ParsePort(elm *etree.Element) (port Port, err error) {
	ret := Port{
		Name: elm.SelectAttrValue("name", ""),
	}

	for _, child := range elm.ChildElements() {
		if child.Tag == "operation" {
			op := PortOperation{
				Name: child.SelectAttrValue("name", ""),
			}

			for _, child := range child.ChildElements() {
				switch child.Tag {
				case "input":
					op.Input.Message = child.SelectAttrValue("message", "")
					op.Input.Name = child.SelectAttrValue("name", "")
				case "output":
					op.Output.Message = child.SelectAttrValue("message", "")
					op.Output.Name = child.SelectAttrValue("name", "")
				case "fault":
					op.Fault.Message = child.SelectAttrValue("message", "")
					op.Fault.Name = child.SelectAttrValue("name", "")
				}
			}

			ret.Operations = append(ret.Operations, op)
		}
	}

	return ret, nil
}

func ParseMesssage(elm *etree.Element, prefixes map[string]string) (msgs Message, err error) {

	ret := Message{
		Name: elm.SelectAttrValue("name", ""),
	}

	for _, child := range elm.ChildElements() {
		if child.Tag == "part" {

			ret.Parts = append(ret.Parts, MessagePart{
				Name:    child.SelectAttrValue("name", ""),
				Element: expandNamespace(child.SelectAttrValue("element", ""), prefixes),
			})
		}
	}

	return ret, nil
}

func expandNamespace(name string, prefixes map[string]string) string {
	ns, n := nsSplit(name)
	ns = prefixes[ns]
	return ns + ":" + n

}

type Schema struct {
	TargetNamespace string
	Elements        []Element
	SubSchemas      []Schema
	Types           []ElementType
}

var parsed = make(map[string]bool)

func ParseSchema(tpelm *etree.Element, targetNamespace string, prefixes map[string]string, source string, depth int) (elements []Element, types []ElementType, err error) {

	key := targetNamespace + source
	if parsed[key] {
		return nil, nil, nil
	}
	parsed[key] = true

	if depth > 10 {
		//log.Println("Recursion depth reached")
		return nil, nil, nil
	}

	for _, child := range tpelm.ChildElements() {

		switch child.Tag {
		case "include":
			loc := child.SelectAttrValue("schemaLocation", "")
			doc, err := getWSDL(loc)
			if err != nil {
				return nil, nil, err
			}
			prefixes := getPrefixToNamespaceMap(doc)

			subElements, subTypes, err := ParseSchema(doc.Root(), tpelm.SelectAttrValue("targetNamespace", targetNamespace), prefixes, loc, depth+1)
			if err != nil {
				return nil, nil, err
			}
			elements = append(elements, subElements...)
			types = append(types, subTypes...)

		case "import":
			loc := child.SelectAttrValue("schemaLocation", "")
			doc, err := getWSDL(loc)
			if err != nil {
				return nil, nil, err
			}
			prefixes := getPrefixToNamespaceMap(doc)
			myTargetNamespace := doc.Root().SelectAttrValue("targetNamespace", targetNamespace)
			subElements, subTypes, err := ParseSchema(doc.Root(), myTargetNamespace, prefixes, loc, depth+1)
			if err != nil {
				return nil, nil, err
			}

			ns := tpelm.SelectAttrValue("namespace", "")
			for _, subElement := range subElements {
				subElement.Name = strings.ReplaceAll(subElement.Name, ns+":", targetNamespace+":")
				elements = append(elements, subElement)
			}
			for _, subType := range subTypes {
				types = append(types, subType)
				subType.Name = strings.ReplaceAll(subType.Name, ns+":", targetNamespace+":")
				types = append(types, subType)
			}

		case "element":
			elm, tp, err := parseElement(child, prefixes, targetNamespace, source)
			if err != nil {
				return nil, nil, err
			}
			elements = append(elements, elm)
			types = append(types, tp...)
		case "simpleType", "complexType":
			tpElm, err := parseTypeElement(child, prefixes, targetNamespace, source)
			if err != nil {
				return nil, nil, err
			}
			types = append(types, tpElm...)
		}

	}

	return elements, types, nil
}

func parseElement(node *etree.Element, prefixes map[string]string, defaultNamespace string, source string) (Element, []ElementType, error) {

	ns, n := nsSplit(node.SelectAttrValue("name", ""))
	rns, rn := nsSplit(node.SelectAttrValue("ref", ""))
	elm := Element{
		NameSpace:          prefixes[ns],
		Name:               n,
		MinOccurs:          parseOccurs(node.SelectAttrValue("minOccurs", "1")),
		MaxOccurs:          parseOccurs(node.SelectAttrValue("maxOccurs", "1")),
		ReferenceNameSpace: prefixes[rns],
		Reference:          rn,
		Source:             source,
	}
	if elm.NameSpace == "" {
		elm.NameSpace = defaultNamespace
	}

	tp := node.SelectAttrValue("type", "")

	if tp != "" {
		// existing type
		elm.ElementType = parseTypeString(tp, prefixes)
		return elm, nil, nil
	}

	var tps []ElementType
	for _, child := range node.ChildElements() {
		switch child.Tag {
		case "simpleType", "complexType":
			tpElm, err := parseTypeElement(child, prefixes, defaultNamespace, source)
			if err != nil {
				return elm, nil, err
			}
			elm.ElementType = tpElm[0].Name
			tps = append(tps, tpElm...)
		}
	}

	return elm, tps, nil
}

var gInternalID = 0

func parseTypeElement(node *etree.Element, prefixes map[string]string, defaultNamespace string, source string) ([]ElementType, error) {

	var ret []ElementType
	tp := ElementType{
		Source: source,
	}

	name := node.SelectAttrValue("name", "")
	if name == "" {
		tp.Name = fmt.Sprintf("internal_%v", gInternalID)
		gInternalID++
	} else {
		ns, n := nsSplit(name)
		if ns == "" {
			ns = defaultNamespace
		}
		tp.Name = n
		tp.NameSpace = ns
	}

	switch node.Tag {
	case "simpleType":

		for _, child2 := range node.ChildElements() {
			if child2.Tag == "restriction" {
				tp.Type = parseTypeString(child2.SelectAttrValue("base", ""), prefixes)
				for _, enum := range child2.ChildElements() {
					if enum.Tag == "enumeration" {
						tp.Enum = append(tp.Enum, enum.SelectAttrValue("value", ""))
					}
					if enum.Tag == "pattern" {
						tp.Pattern = enum.SelectAttrValue("value", "")
					}
				}
			}
		}
	case "complexType":
		for _, child2 := range node.ChildElements() {
			if child2.Tag == "annotation" {
				continue
			}
			if child2.Tag == "sequence" {
				subElm, tps, err := parseElement(child2, prefixes, defaultNamespace, source)
				if err != nil {
					return ret, err
				}
				tp.SubElements = append(tp.SubElements, subElm)
				ret = append(ret, tps...)
			}
		}
	}

	ret = append([]ElementType{tp}, ret...)
	return ret, nil
}

func parseTypeString(tp string, prefixes map[string]string) string {
	ns, n := nsSplit(tp)
	ns = prefixes[ns]
	return ns + ":" + n
}

func parseOccurs(a string) int {
	if a == "" {
		return 1
	}

	r, _ := strconv.Atoi(a)
	return r
}

type Element struct {
	Source             string
	NameSpace          string
	Name               string
	ElementType        string
	MinOccurs          int
	MaxOccurs          int
	ReferenceNameSpace string
	Reference          string
}

func (e Element) FullName() string {
	return e.NameSpace + ":" + e.Name
}

type ElementType struct {
	Source      string
	Type        string
	NameSpace   string
	Name        string
	SubElements []Element
	Enum        []string
	Pattern     string
}

func (e ElementType) FullName() string {
	return e.NameSpace + ":" + e.Name
}
