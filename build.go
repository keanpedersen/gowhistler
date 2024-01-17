package gowhistler

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
)


type goTypeName struct {
	Type string
	TODO: xml-tag vars
}

type Builder struct {
	Types map[string]string
	Vars  map[string]string
}

func (b *Builder) OutputTypes(w io.Writer) {
	for name, def := range b.Types {
		fmt.Fprintf(w, "type %s %s\n", name, def)
	}

	fmt.Fprintf(w, "\n\n")
	for name, tp := range b.Vars {
		fmt.Fprintf(w, "var %s %s\n", name, tp)
	}
}

func (wsdl *WSDL) Build() error {

	builder := &Builder{
		Types: make(map[string]string),
		Vars:  make(map[string]string),
	}

	for _, message := range wsdl.Messages {
		if err := wsdl.BuildMessage(builder, message); err != nil {
			return err
		}
	}

	for _, service := range wsdl.Services {
		if err := wsdl.BuildService(service); err != nil {
			return err
		}
	}

	f, err := os.Create("output/struct.go")
	if err != nil {
		return errors.WithStack(err)
	}

	f.WriteString("package output\n\n")
	f.WriteString("import \"time\"\n\n")

	builder.OutputTypes(f)

	f.Close()

	return nil
}

func (wsdl *WSDL) BuildMessage(builder *Builder, message Message) error {

	fmt.Printf("Building message %v\n", message.Name)

	for _, part := range message.Parts {

		fmt.Printf("Building part %v of type %v\n", part.Name, part.Element)

		tp, ok := wsdl.TypeMap[strings.ToLower(part.Element)]
		if !ok {
			return errors.Errorf("Could not find element of type %v", part.Element)
		}

		if err := wsdl.BuildType(builder, tp); err != nil {
			return err
		}

		builder.Vars[ucFirst(message.Name+"_"+part.Name)] = tp.TypeName()

	}

	return nil
}

func (wsdl *WSDL) BuildType(builder *Builder, tp ElementType) error {

	if tp.BuildIn != "" {
		//builder.Types[tp.TypeName()] = tp.BuildIn
		return nil
	}

	thisType := ""
	if len(tp.SubElements) > 0 || len(tp.ChoiceElements) > 0 || len(tp.AttributeElements) > 0 {
		thisType = "struct {\n"

		for _, sub := range tp.SubElements {
			if sub.Reference != "" {
				subTp, ok := wsdl.TypeMap[strings.ToLower(sub.ReferenceNameSpace+":"+sub.Reference)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v:%v", sub.ReferenceNameSpace, sub.Reference)
				}

				thisType += fmt.Sprintf("%s %s `xml:\"%s\"`\n", makeTypeName(fmt.Sprintf("%v__%v ", sub.ReferenceNameSpace, sub.Reference)), subTp.TypeName(), sub.Reference)

				if err := wsdl.BuildType(builder, subTp); err != nil {
					return err
				}
			} else {
				subTpName := sub.ElementType
				if !strings.Contains(sub.ElementType, ":") {
					subTpName = sub.NameSpace + ":" + subTpName
				}
				subTp, ok := wsdl.TypeMap[strings.ToLower(subTpName)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v", subTpName)
				}
				if err := wsdl.BuildType(builder, subTp); err != nil {
					return err
				}
				thisType += fmt.Sprintf("%s %s\n", sub.Name, subTp.TypeName())
			}
		}
		for _, sub := range tp.ChoiceElements {
			if sub.Reference != "" {
				subTp, ok := wsdl.TypeMap[strings.ToLower(sub.ReferenceNameSpace+":"+sub.Reference)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v:%v", sub.ReferenceNameSpace, sub.Reference)
				}

				thisType += fmt.Sprintf("%s *%s `xml:\"%s\"`\n", makeTypeName(fmt.Sprintf("%v__%v ", sub.ReferenceNameSpace, sub.Reference)), subTp.TypeName(), sub.Reference)

				if err := wsdl.BuildType(builder, subTp); err != nil {
					return err
				}
			} else {
				subTpName := sub.ElementType
				if !strings.Contains(sub.ElementType, ":") {
					subTpName = sub.NameSpace + ":" + subTpName
				}
				subTp, ok := wsdl.TypeMap[strings.ToLower(subTpName)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v", subTpName)
				}
				if err := wsdl.BuildType(builder, subTp); err != nil {
					return err
				}
				thisType += fmt.Sprintf("%s *%s\n", sub.Name, subTp.TypeName())
			}
		}

		for name, subTpName := range tp.AttributeElements {

			subTp, ok := wsdl.TypeMap[strings.ToLower(subTpName)]
			if !ok {
				return errors.Errorf("Could not find attribute reference of type %v", subTpName)
			}
			if err := wsdl.BuildType(builder, subTp); err != nil {
				return err
			}
			thisType += fmt.Sprintf("%s %s `xml:\",attr\"`\n", ucFirst(name), subTp.TypeName())

		}

		thisType += "}"
	} else if tp.Type != "" {
		subTp, ok := wsdl.TypeMap[strings.ToLower(tp.Type)]
		if !ok {
			return errors.Errorf("Could not find reference of type %v", tp.Type)
		}

		if subTp.BuildIn != "" {
			thisType = subTp.BuildIn
		} else {
			if err := wsdl.BuildType(builder, subTp); err != nil {
				return err
			}
		}
	}

	builder.Types[tp.TypeName()] = thisType

	return nil
}

func (wsdl *WSDL) BuildService(service Service) error {

	fmt.Printf("Building service %v\n", service.Name)

	for _, port := range service.Ports {

		binding, err := wsdl.FindBinding(port.Binding)
		if err != nil {
			return err
		}

		fmt.Printf("Binding %s found for port %s\n", binding.Name, port.Name)

		fmt.Printf("Port %s is at address %s\n", port.Name, port.AddressLocation)

		for _, op := range binding.Operations {
			if err := wsdl.BuildOperation(op); err != nil {
				return err
			}
		}
	}

	return nil
}

func (wsdl *WSDL) BuildOperation(op BindingOperation) error {

	fmt.Printf("Building operation %v at %v\n", op.Name, op.SoapAction)

	for _, component := range op.Input {
		if err := wsdl.BuildOperationComponent(component, "input"); err != nil {
			return err
		}
	}
	for _, component := range op.Output {
		if err := wsdl.BuildOperationComponent(component, "output"); err != nil {
			return err
		}
	}
	for _, component := range op.Fault {
		if err := wsdl.BuildOperationComponent(component, "fault"); err != nil {
			return err
		}
	}

	return nil
}

func (wsdl *WSDL) BuildOperationComponent(component BindingOperationComponent, tp string) error {

	fmt.Printf("Building %v operation component in %v\n", tp, component.In)

	return nil
}

func (wsdl *WSDL) FindBinding(name string) (binding Binding, err error) {

	_, n := nsSplit(name)

	for _, binding := range wsdl.Bindings {
		if binding.Name == n {
			return binding, nil
		}
	}

	return binding, errors.Errorf("Binding not found: %s", name)
}

func (wsdl *WSDL) GetTargetNamespace() (targetNS, targetNSPrefix string) {
	return wsdl.TargetNamespace, wsdl.UrlToNameSpaceMapping[wsdl.TargetNamespace]
}

func nsSplit(n string) (ns, name string) {

	parts := strings.SplitN(n, ":", 2)
	if len(parts) == 1 {
		return "", n
	}

	return parts[0], parts[1]
}
