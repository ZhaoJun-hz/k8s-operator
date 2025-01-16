/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1 "deployment/api/v1"
)

// nolint:unused
// log is for logging in this package.
var mydeploymentlog = logf.Log.WithName("mydeployment-resource")

// SetupMyDeploymentWebhookWithManager registers the webhook for MyDeployment in the manager.
func SetupMyDeploymentWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&appsv1.MyDeployment{}).
		WithValidator(&MyDeploymentCustomValidator{}).
		WithDefaulter(&MyDeploymentCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-apps-shudong-com-v1-mydeployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.shudong.com,resources=mydeployments,verbs=create;update,versions=v1,name=mmydeployment-v1.kb.io,admissionReviewVersions=v1

// MyDeploymentCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind MyDeployment when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type MyDeploymentCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &MyDeploymentCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MyDeployment.
func (d *MyDeploymentCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	mydeployment, ok := obj.(*appsv1.MyDeployment)

	if !ok {
		return fmt.Errorf("expected an MyDeployment object but got %T", obj)
	}
	mydeploymentlog.Info("Defaulting for MyDeployment", "name", mydeployment.GetName())

	if mydeployment.Spec.Replicas == 0 {
		// 不能确定用户给定的服务是否是一个无状态的应用，如果是有状态的，
		// 多个副本会造成数据错乱，所以保守的，只给一个副本
		mydeployment.Spec.Replicas = 1
	}

	// 可以允许用户自己指定 service 的 port 值
	// 如果不指定，则使用服务的 port 值来代替
	if mydeployment.Spec.Expose.ServicePort == 0 {
		mydeployment.Spec.Expose.ServicePort = mydeployment.Spec.Port
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-apps-shudong-com-v1-mydeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.shudong.com,resources=mydeployments,verbs=create;update,versions=v1,name=vmydeployment-v1.kb.io,admissionReviewVersions=v1

// MyDeploymentCustomValidator struct is responsible for validating the MyDeployment resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type MyDeploymentCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &MyDeploymentCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type MyDeployment.
func (v *MyDeploymentCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mydeployment, ok := obj.(*appsv1.MyDeployment)
	if !ok {
		return nil, fmt.Errorf("expected a MyDeployment object but got %T", obj)
	}
	mydeploymentlog.Info("Validation for MyDeployment upon creation", "name", mydeployment.GetName())

	return nil, mydeployment.ValidateCreateAndUpdate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type MyDeployment.
func (v *MyDeploymentCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	mydeployment, ok := newObj.(*appsv1.MyDeployment)
	if !ok {
		return nil, fmt.Errorf("expected a MyDeployment object for the newObj but got %T", newObj)
	}
	mydeploymentlog.Info("Validation for MyDeployment upon update", "name", mydeployment.GetName())

	return nil, mydeployment.ValidateCreateAndUpdate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type MyDeployment.
func (v *MyDeploymentCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mydeployment, ok := obj.(*appsv1.MyDeployment)
	if !ok {
		return nil, fmt.Errorf("expected a MyDeployment object but got %T", obj)
	}
	mydeploymentlog.Info("Validation for MyDeployment upon deletion", "name", mydeployment.GetName())

	return nil, nil
}
