package controller

import (
	"bytes"
	myApiV1 "deployment/api/v1"
	"fmt"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"text/template"
)

func parseTemplate(myDeployment *myApiV1.MyDeployment, templateName string) ([]byte, error) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("templates/%s", templateName))
	if err != nil {
		return nil, err
	}
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, myDeployment)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func NewDeployment(myDeployment *myApiV1.MyDeployment) (*appsV1.Deployment, error) {
	content, err := parseTemplate(myDeployment, "deployment.yaml")
	if err != nil {
		return nil, err
	}
	deploy := new(appsV1.Deployment)

	err = yaml.Unmarshal(content, deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func NewIngress(myDeployment *myApiV1.MyDeployment) (*networkingV1.Ingress, error) {
	content, err := parseTemplate(myDeployment, "ingress.yaml")
	if err != nil {
		return nil, err
	}
	ingress := new(networkingV1.Ingress)

	err = yaml.Unmarshal(content, ingress)
	if err != nil {
		return nil, err
	}
	return ingress, nil
}

func NewService(myDeployment *myApiV1.MyDeployment) (*coreV1.Service, error) {
	content, err := parseTemplate(myDeployment, "service.yaml")
	if err != nil {
		return nil, err
	}
	service := new(coreV1.Service)

	err = yaml.Unmarshal(content, service)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func NewNodePortService(myDeployment *myApiV1.MyDeployment) (*coreV1.Service, error) {
	content, err := parseTemplate(myDeployment, "service-nodeport.yaml")
	if err != nil {
		return nil, err
	}
	service := new(coreV1.Service)

	err = yaml.Unmarshal(content, service)
	if err != nil {
		return nil, err
	}
	return service, nil
}
