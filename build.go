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

		fmt.Printf("Building part %v of type %v\n", part.Name, part.Element)

		element, ok := wsdl.TypeMap[part.Element]
		if !ok {
			return errors.Errorf("Could not find element of type %v", part.Element)
		}

		if err := wsdl.BuildType(element); err != nil {
			return err
		}

	}

	return nil
}

func (wsdl *WSDL) BuildType(element ElementType) error {

	fmt.Printf("Building element %s %v\n", element.Type, element.FullName())
	/*
		switch element.Type {
		case "complex":
			for _, subElement := range element.SubElements {
				if err := wsdl.BuildElement(subElement); err != nil {
					return err
				}
			}
		case "reference":
			elmName := element.ReferenceNameSpace + ":" + element.Reference
			elm, ok := wsdl.TypeMap[elmName]
			if !ok {
				return errors.Errorf("Could not find element of type %v", elmName)
			}
			if err := wsdl.BuildElement(elm); err != nil {
				return err
			}
		}
	*/
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
