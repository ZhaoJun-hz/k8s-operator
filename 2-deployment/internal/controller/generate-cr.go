package controller

import (
	"bytes"
	myApiV1 "deployment/api/v1"
	"fmt"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"text/template"
)

func parseTemplate(myDeployment *myApiV1.MyDeployment, templateName string) ([]byte, error) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("internal/controller/templates/%s", templateName))
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
	var content []byte
	var err error
	// https 6. ingress 添加 tls 支持
	if myDeployment.Spec.Expose.Mode == myApiV1.ModeIngress && myDeployment.Spec.Expose.Tls {
		content, err = parseTemplate(myDeployment, "ingress-with-tls.yaml")
	} else {
		content, err = parseTemplate(myDeployment, "ingress.yaml")
	}
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

func NewIssuer(myDeployment *myApiV1.MyDeployment) (*unstructured.Unstructured, error) {
	if myDeployment.Spec.Expose.Mode != myApiV1.ModeIngress || !myDeployment.Spec.Expose.Tls {
		return nil, nil
	}
	//apiVersion: cert-manager.io/v1
	//kind: Issuer
	//metadata:
	//  name: selfsigned-issuer
	//  namespace: system
	//spec:
	//  selfSigned: {}
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Issuer",
			"metadata": map[string]interface{}{
				"name":      myDeployment.Name,
				"namespace": myDeployment.Namespace,
			},
			"spec": map[string]interface{}{
				"selfSigned": map[string]interface{}{},
			},
		},
	}, nil
}

func NewCertificate(myDeployment *myApiV1.MyDeployment) (*unstructured.Unstructured, error) {
	if myDeployment.Spec.Expose.Mode != myApiV1.ModeIngress || !myDeployment.Spec.Expose.Tls {
		return nil, nil
	}
	/*
		apiVersion: cert-manager.io/v1
		kind: Certificate
		metadata:
		  name: serving-cert
		  namespace: system
		spec:
		  dnsNames:
		  - <spec.expose.ingressDomain>
		  issuerRef:
		    kind: Issuer
		    name: selfsigned-issuer
		  secretName: webhook-server-cert
	*/
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"metadata": map[string]interface{}{
				"name":      myDeployment.Name,
				"namespace": myDeployment.Namespace,
			},
			"spec": map[string]interface{}{
				"dnsNames": []interface{}{
					myDeployment.Spec.Expose.IngressDomain,
				},
				"issuerRef": map[string]interface{}{
					"kind": "Issuer",
					"name": myDeployment.Name,
				},
				"secretName": myDeployment.Name,
			},
		},
	}, nil
}
