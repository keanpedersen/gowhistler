package gowhistler

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

func (wsdl *WSDL) Build() error {

	for _, message := range wsdl.Messages {
		if err := wsdl.BuildMessage(message); err != nil {
			return err
		}
	}

	for _, service := range wsdl.Services {
		if err := wsdl.BuildService(service); err != nil {
			return err
		}
	}

	return nil
}

func (wsdl *WSDL) BuildMessage(message Message) error {

	fmt.Printf("Building message %v\n", message.Name)

	for _, part := range message.Parts {

		fmt.Printf("\n\nBuilding part %v of type %v\n\n", part.Name, part.Element)

		tp, ok := wsdl.TypeMap[strings.ToLower(part.Element)]
		if !ok {
			return errors.Errorf("Could not find element of type %v", part.Element)
		}

		fmt.Printf("type %v ", makeTypeName(part.Element))

		if err := wsdl.BuildType(tp); err != nil {
			return err
		}

		fmt.Printf("var %v %v\n", part.Name, makeTypeName(part.Element))

	}

	return nil
}

func (wsdl *WSDL) BuildType(tp ElementType) error {

	if tp.BuildIn != "" {
		fmt.Println(tp.BuildIn)
		return nil
	}

	if len(tp.SubElements) > 0 {
		fmt.Printf("struct {\n")

		for _, sub := range tp.SubElements {
			if sub.Reference != "" {
				subTp, ok := wsdl.TypeMap[strings.ToLower(sub.ReferenceNameSpace+":"+sub.Reference)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v:%v", sub.ReferenceNameSpace, sub.Reference)
				}
				fmt.Printf(makeTypeName(fmt.Sprintf("%v__%v ", sub.ReferenceNameSpace, sub.Reference)))
				if err := wsdl.BuildType(subTp); err != nil {
					return err
				}
			} else {
				fmt.Printf("%v ", sub.Name)
				subTpName := sub.ElementType
				if !strings.Contains(sub.ElementType, ":") {
					subTpName = sub.NameSpace + ":" + subTpName
				}
				subTp, ok := wsdl.TypeMap[strings.ToLower(subTpName)]
				if !ok {
					return errors.Errorf("Could not find reference of type %v", subTpName)
				}
				if err := wsdl.BuildType(subTp); err != nil {
					return err
				}
			}

		}
		fmt.Println("}")
	} else if tp.Type != "" {
		subTp, ok := wsdl.TypeMap[strings.ToLower(tp.Type)]
		if !ok {
			return errors.Errorf("Could not find reference of type %v", tp.Type)
		}
		if err := wsdl.BuildType(subTp); err != nil {
			return err
		}
	}

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
