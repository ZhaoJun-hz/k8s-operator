package controller

import (
	"bytes"
	myApiV1 "deployment/api/v1"
	"fmt"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
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

var IngressClassName = "nginx"
var PrefixPathType = networkingV1.PathTypePrefix

func newLabels(myDeployment *myApiV1.MyDeployment) map[string]string {
	return map[string]string{
		"app": myDeployment.Name,
	}
}

func NewDeployment(myDeployment *myApiV1.MyDeployment) appsV1.Deployment {
	// 1. 创建基本的 deployment
	// 1.1 创建只含有 metadata 的信息对象
	deploy := newBaseDeployment(myDeployment)

	// 2. 创建附加的对象
	// 2.1 在基本的 deployment 中添加其他的对象
	deploy.Spec.Replicas = &myDeployment.Spec.Replicas
	deploy.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: newLabels(myDeployment),
	}
	deploy.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name:   myDeployment.Name,
		Labels: newLabels(myDeployment),
	}
	deploy.Spec.Template.Spec.Containers = []coreV1.Container{
		newBaseContainer(myDeployment),
	}
	return deploy
}

func newBaseContainer(myDeployment *myApiV1.MyDeployment) coreV1.Container {
	c := coreV1.Container{
		Name:  myDeployment.ObjectMeta.Name,
		Image: myDeployment.Spec.Image,
		Ports: []coreV1.ContainerPort{
			{
				ContainerPort: myDeployment.Spec.Port,
			},
		},
	}
	if len(myDeployment.Spec.StartCmd) != 0 {
		c.Command = myDeployment.Spec.StartCmd
	}

	if len(myDeployment.Spec.Args) != 0 {
		c.Args = myDeployment.Spec.Args
	}

	if len(myDeployment.Spec.Environments) != 0 {
		c.Env = myDeployment.Spec.Environments
	}

	return c
}

func newBaseDeployment(myDeployment *myApiV1.MyDeployment) appsV1.Deployment {
	return appsV1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      myDeployment.Name,
			Namespace: myDeployment.Namespace,
			Labels:    newLabels(myDeployment),
		},
	}
}

func NewIngress(myDeployment *myApiV1.MyDeployment) networkingV1.Ingress {
	ingress := newBaseIngress(myDeployment)
	ingress.Spec.IngressClassName = &IngressClassName
	ingress.Spec.Rules = []networkingV1.IngressRule{
		newIngressRule(myDeployment),
	}
	// https 6. ingress 添加 tls 支持
	if myDeployment.Spec.Expose.Mode == myApiV1.ModeIngress && myDeployment.Spec.Expose.Tls {
		ingress.Spec.TLS = []networkingV1.IngressTLS{
			newIngressTLS(myDeployment),
		}
	}

	return ingress

	//var content []byte
	//var err error
	// https 6. ingress 添加 tls 支持
	//if myDeployment.Spec.Expose.Mode == myApiV1.ModeIngress && myDeployment.Spec.Expose.Tls {
	//	content, err = parseTemplate(myDeployment, "ingress-with-tls.yaml")
	//} else {
	//	content, err = parseTemplate(myDeployment, "ingress.yaml")
	//}
	//if err != nil {
	//	return nil, err
	//}
	//ingress := new(networkingV1.Ingress)
	//
	//err = yaml.Unmarshal(content, ingress)
	//if err != nil {
	//	return nil, err
	//}
	//return ingress, nil
}

func newBaseIngress(deployment *myApiV1.MyDeployment) networkingV1.Ingress {
	return networkingV1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
	}
}

func newIngressRule(deployment *myApiV1.MyDeployment) networkingV1.IngressRule {
	return networkingV1.IngressRule{
		Host: deployment.Spec.Expose.IngressDomain,
		IngressRuleValue: networkingV1.IngressRuleValue{
			HTTP: &networkingV1.HTTPIngressRuleValue{
				Paths: []networkingV1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: &PrefixPathType,
						Backend: networkingV1.IngressBackend{
							Service: &networkingV1.IngressServiceBackend{
								Name: deployment.Name,
								Port: networkingV1.ServiceBackendPort{
									Number: int32(deployment.Spec.Expose.ServicePort),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newIngressTLS(deployment *myApiV1.MyDeployment) networkingV1.IngressTLS {
	return networkingV1.IngressTLS{
		Hosts:      []string{deployment.Spec.Expose.IngressDomain},
		SecretName: deployment.Name,
	}
}

func NewService(myDeployment *myApiV1.MyDeployment) coreV1.Service {
	svc := newBaseService(myDeployment)
	svc.Spec.Selector = newLabels(myDeployment)

	servicePort := newServicePort(myDeployment)

	switch myDeployment.Spec.Expose.Mode {
	case myApiV1.ModeIngress:
		svc.Spec.Ports = []coreV1.ServicePort{servicePort}
	case myApiV1.ModeNodePort:
		svc.Spec.Type = coreV1.ServiceTypeNodePort
		servicePort.NodePort = myDeployment.Spec.Expose.NodePort
		svc.Spec.Ports = []coreV1.ServicePort{servicePort}
	default:
		return coreV1.Service{}
	}
	return svc
}

func newBaseService(deployment *myApiV1.MyDeployment) coreV1.Service {
	return coreV1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
	}
}

func newServicePort(deployment *myApiV1.MyDeployment) coreV1.ServicePort {
	return coreV1.ServicePort{
		Protocol: coreV1.ProtocolTCP,
		Port:     deployment.Spec.Expose.ServicePort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: deployment.Spec.Expose.ServicePort,
		},
	}
}

//func NewNodePortService(myDeployment *myApiV1.MyDeployment) (*coreV1.Service, error) {
//	content, err := parseTemplate(myDeployment, "service-nodeport.yaml")
//	if err != nil {
//		return nil, err
//	}
//	service := new(coreV1.Service)
//
//	err = yaml.Unmarshal(content, service)
//	if err != nil {
//		return nil, err
//	}
//	return service, nil
//}

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
