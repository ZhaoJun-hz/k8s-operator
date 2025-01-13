package controller

import (
	myApiV1 "deployment/api/v1"
	"fmt"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"reflect"
	"testing"
)

func readFile(path string) []byte {
	content, err := os.ReadFile(fmt.Sprintf("testdata/%s", path))
	if err != nil {
		panic(err)
	}
	return content
}

func newMyDeployment(filename string) *myApiV1.MyDeployment {
	content := readFile(filename)
	myDeployment := new(myApiV1.MyDeployment)
	err := yaml.Unmarshal(content, myDeployment)
	if err != nil {
		panic(err)
	}
	return myDeployment
}

func newDeployment(filename string) *appsV1.Deployment {
	content := readFile(filename)
	deployment := new(appsV1.Deployment)
	err := yaml.Unmarshal(content, deployment)
	if err != nil {
		panic(err)
	}
	return deployment
}

func newService(filename string) *coreV1.Service {
	content := readFile(filename)
	service := new(coreV1.Service)
	err := yaml.Unmarshal(content, service)
	if err != nil {
		panic(err)
	}
	return service
}

func newIngress(filename string) *networkingV1.Ingress {
	content := readFile(filename)
	ingress := new(networkingV1.Ingress)
	err := yaml.Unmarshal(content, ingress)
	if err != nil {
		panic(err)
	}
	return ingress
}

func TestNewDeployment(t *testing.T) {
	type args struct {
		myDeployment *myApiV1.MyDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *appsV1.Deployment
		wantErr bool
	}{
		{
			name: "测试使用 ingress mode，生成 Deployment 资源",
			args: args{
				myDeployment: newMyDeployment("ingress-cr.yaml"),
			},
			want:    newDeployment("ingress-deployment-expect.yaml"),
			wantErr: false,
		},
		{
			name: "测试使用 nodePort mode，生成 Deployment 资源",
			args: args{
				myDeployment: newMyDeployment("nodeport-cr.yaml"),
			},
			want:    newDeployment("nodeport-deployment-expect.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDeployment(tt.args.myDeployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeployment() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewIngress(t *testing.T) {
	type args struct {
		myDeployment *myApiV1.MyDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *networkingV1.Ingress
		wantErr bool
	}{
		{
			name: "测试使用 ingress mode，生成 Ingress 资源",
			args: args{
				myDeployment: newMyDeployment("ingress-cr.yaml"),
			},
			want:    newIngress("ingress-ingress-expect.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIngress(tt.args.myDeployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIngress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIngress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNodePortService(t *testing.T) {
	type args struct {
		myDeployment *myApiV1.MyDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *coreV1.Service
		wantErr bool
	}{
		{
			name: "测试使用 nodePort mode，生成 NodePort Service 资源",
			args: args{
				myDeployment: newMyDeployment("nodeport-cr.yaml"),
			},
			want:    newService("nodeport-service-expect.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNodePortService(tt.args.myDeployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNodePortService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNodePortService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	type args struct {
		myDeployment *myApiV1.MyDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *coreV1.Service
		wantErr bool
	}{
		{
			name: "测试使用 ingress mode，生成 Service 资源",
			args: args{
				myDeployment: newMyDeployment("ingress-cr.yaml"),
			},
			want:    newService("ingress-service-expect.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewService(tt.args.myDeployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewService() got = %v, want %v", got, tt.want)
			}
		})
	}
}
