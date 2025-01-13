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

package controller

import (
	"context"
	myApiV1 "deployment/api/v1"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MyDeploymentReconciler reconciles a MyDeployment object
type MyDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *MyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "MyDeployment", req.NamespacedName)
	logger.Info("Starting MyDeployment Reconcile")
	// 1. 获取资源对象
	myDeployment := new(myApiV1.MyDeployment)
	err := r.Get(ctx, req.NamespacedName, myDeployment)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// 防止污染缓存
	myDeploymentCopy := myDeployment.DeepCopy()
	// ============ 处理 deployment ===============
	// 2. 获取 deployment 资源对象
	deployment := new(appsV1.Deployment)
	err = r.Get(ctx, req.NamespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// 2.1 不存在对象
			// 2.1.1 创建 deployment
			err := r.createDeployment(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// 2.2 存在对象
		// 2.2.1 更新 deployment
		err := r.updateDeployment(ctx, myDeploymentCopy)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// ============ 处理 service ===============
	// 3. 获取 service 资源对象
	service := new(coreV1.Service)
	err = r.Get(ctx, req.NamespacedName, service)
	if err != nil {
		if errors.IsNotFound(err) {
			// 3.1 不存在对象
			// 3.1.1 mode 为 ingress
			if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeIngress {
				// 3.1.1.1 创建普通 service
				err := r.createService(ctx, myDeploymentCopy)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
				// 3.1.2 mode 为 nodePort
				// 3.1.2.1 创建 nodePort 模式的 service
				err := r.createNodePortService(ctx, myDeploymentCopy)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{}, myApiV1.ErrorNotSupportedMode
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// 3.2 存在对象
		// 3.2.1 mode 为 ingress
		if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeIngress {
			// 3.2.1.1 更新普通 service
			err := r.updateService(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
			// 3.2.2 mode 为 nodePort
			// 3.2.2.1 更新 nodePort 模式的 service
			err := r.updateNodePortService(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, myApiV1.ErrorNotSupportedMode
		}

	}

	// ============ 处理 ingress ===============
	// 4. 获取 ingress 资源对象
	ingress := new(networkingV1.Ingress)
	err = r.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			// 4.1 不存在对象
			// 4.1.1 mode 为 ingress
			if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeIngress {
				// 4.1.1.1 创建 ingress
				err := r.createIngress(ctx, myDeploymentCopy)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
				// 4.1.2 mode 为 nodePort
				// 4.1.2.1 退出
				return ctrl.Result{}, nil
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// 4.2 存在对象
		if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeIngress {
			// 4.2.1 mode 为 ingress
			// 4.2.1.1 更新 ingress
			err := r.updateIngress(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
			// 4.2.2 mode 为 nodePort
			// 4.2.2.1 删除 ingress
			err := r.deleteIngress(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	logger.Info("End MyDeployment Reconcile")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myApiV1.MyDeployment{}).
		Named("mydeployment").
		Complete(r)
}

func (r *MyDeploymentReconciler) createDeployment(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	deployment, err := NewDeployment(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, deployment)
}

func (r *MyDeploymentReconciler) updateDeployment(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	deployment, err := NewDeployment(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Update(ctx, deployment)
}

func (r *MyDeploymentReconciler) createService(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	service, err := NewService(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, service)
}

func (r *MyDeploymentReconciler) createNodePortService(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	service, err := NewNodePortService(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, service)
}

func (r *MyDeploymentReconciler) updateService(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	service, err := NewService(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Update(ctx, service)
}

func (r *MyDeploymentReconciler) updateNodePortService(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	service, err := NewNodePortService(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Update(ctx, service)
}

func (r *MyDeploymentReconciler) createIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	ingress, err := NewIngress(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, ingress)
}

func (r *MyDeploymentReconciler) updateIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	ingress, err := NewIngress(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Update(ctx, ingress)
}

func (r *MyDeploymentReconciler) deleteIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	ingress, err := NewIngress(myDeployment)
	if err != nil {
		return err
	}
	return r.Client.Delete(ctx, ingress)
}
