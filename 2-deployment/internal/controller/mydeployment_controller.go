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
	"fmt"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var WaitRequest = 10 * time.Second

// MyDeploymentReconciler reconciles a MyDeployment object
type MyDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// 用来访问 issuer 和 certificate 资源
	DynamicClient dynamic.Interface
}

// https 2. 创建动态 GVR
var (
	// https 2.1 issuer GVR，供 DynamicClient 调用
	issuerGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "issuers",
	}
	// https 2.2 certificate GVR，供 DynamicClient 调用
	certificateGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "certificates",
	}
)

// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.shudong.com,resources=mydeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// https 3. 创建 issuer certificate GVR 需要的权限
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch

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
	// 状态更新策略
	// 创建的时候
	//		更新为创建
	// 更新的时候
	//		根据获取的状态 来判断是否更新 status
	// 删除的时候
	//		只有在 操作 Ingress 的时候，并且 mode 为 nodePort 的时候

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

	// 处理最终的返回
	defer func() {
		if r.Ready(myDeploymentCopy) {
			_ = r.Client.Status().Update(ctx, myDeploymentCopy)
			return
		}
		if myDeploymentCopy.Status.ObservedGeneration != myDeployment.Status.ObservedGeneration {
			_ = r.Client.Status().Update(ctx, myDeploymentCopy)
		}
	}()

	// ============ 处理 deployment ===============
	// 2. 获取 deployment 资源对象
	deployment := new(appsV1.Deployment)
	err = r.Get(ctx, req.NamespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// 2.1 不存在对象
			// 2.1.1 创建 deployment
			errCreate := r.createDeployment(ctx, myDeploymentCopy)
			if errCreate != nil {
				return ctrl.Result{}, errCreate
			}
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeDeployment,
				fmt.Sprintf(myApiV1.ConditionMessageDeploymentNotOKFmt, req.Name),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonDeploymentNotReady)
		} else {
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeDeployment,
				fmt.Sprintf("Deployment %s, err: %s", req.Name, err.Error()),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonDeploymentNotReady)
			return ctrl.Result{}, err
		}
	} else {
		// 2.2 存在对象
		// 2.2.1 更新 deployment
		err := r.updateDeployment(ctx, myDeploymentCopy, deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		if *deployment.Spec.Replicas == deployment.Status.ReadyReplicas {
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeDeployment,
				fmt.Sprintf(myApiV1.ConditionMessageDeploymentOKFmt, req.Name),
				myApiV1.ConditionStatusTrue, myApiV1.ConditionReasonDeploymentReady)
		} else {
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeDeployment,
				fmt.Sprintf(myApiV1.ConditionMessageDeploymentNotOKFmt, req.Name),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonDeploymentNotReady)

		}

	}

	// ============ 处理 service ===============
	// 3. 获取 service 资源对象
	service := new(coreV1.Service)
	err = r.Get(ctx, req.NamespacedName, service)
	if err != nil {
		if errors.IsNotFound(err) {
			// 3.1 不存在对象, 创建 service
			err := r.createService(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}

			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeService,
				fmt.Sprintf(myApiV1.ConditionMessageServiceNotOKFmt, req.Name),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonServiceNotReady)
		} else {
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeService,
				fmt.Sprintf("Service %s, err: %s", req.Name, err.Error()),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonServiceNotReady)
			return ctrl.Result{}, err
		}
	} else {
		// 3.2 存在对象，更新 service
		err := r.updateService(ctx, myDeploymentCopy, service)
		if err != nil {
			return ctrl.Result{}, err
		}

		r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeService,
			fmt.Sprintf(myApiV1.ConditionMessageServiceOKFmt, req.Name),
			myApiV1.ConditionStatusTrue, myApiV1.ConditionReasonServiceReady)
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
				r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeIngress,
					fmt.Sprintf(myApiV1.ConditionMessageIngressNotOKFmt, req.Name),
					myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonIngressNotReady)
				if myDeploymentCopy.Spec.Expose.Tls {
					// https 4. 创建 issuers 和 certificate
					err := r.createIssuer(ctx, myDeploymentCopy)
					if err != nil {
						return ctrl.Result{}, err
					}
					err = r.createCertificate(ctx, myDeploymentCopy)
					if err != nil {
						return ctrl.Result{}, err
					}
				}
			} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
				// 4.1.2 mode 为 nodePort
				// 4.1.2.1 退出
				return ctrl.Result{}, nil
			}
		} else {
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeIngress,
				fmt.Sprintf("Ingress %s, err: %s", req.Name, err.Error()),
				myApiV1.ConditionStatusFalse, myApiV1.ConditionReasonIngressNotReady)
			return ctrl.Result{}, err
		}
	} else {
		// 4.2 存在对象
		if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeIngress {
			// 4.2.1 mode 为 ingress
			// 4.2.1.1 更新 ingress
			err := r.updateIngress(ctx, myDeploymentCopy, ingress)
			if err != nil {
				return ctrl.Result{}, err
			}
			r.updateConditions(myDeploymentCopy, myApiV1.ConditionTypeIngress,
				fmt.Sprintf(myApiV1.ConditionMessageIngressOKFmt, req.Name),
				myApiV1.ConditionStatusTrue, myApiV1.ConditionReasonIngressReady)
			// https 5. 创建 issuers 和 certificate
			if myDeploymentCopy.Spec.Expose.Tls {
				// https 4. 创建 issuers 和 certificate
				err := r.createIssuer(ctx, myDeploymentCopy)
				if err != nil {
					return ctrl.Result{}, err
				}
				err = r.createCertificate(ctx, myDeploymentCopy)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		} else if myDeploymentCopy.Spec.Expose.Mode == myApiV1.ModeNodePort {
			// 4.2.2 mode 为 nodePort
			// 4.2.2.1 删除 ingress
			err := r.deleteIngress(ctx, myDeploymentCopy)
			if err != nil {
				return ctrl.Result{}, err
			}
			r.deleteStatus(myDeploymentCopy, myApiV1.ConditionTypeIngress)
		}
	}

	logger.Info("End MyDeployment Reconcile")
	if !r.Ready(myDeploymentCopy) {
		return ctrl.Result{RequeueAfter: WaitRequest}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myApiV1.MyDeployment{}).
		// 监控 Deployment 类型，变更就触发 Reconcile 方法的执行
		Owns(&appsV1.Deployment{}).
		// 监控 Service 类型，变更就触发 Reconcile 方法的执行
		Owns(&coreV1.Service{}).
		// 监控 Ingress 类型，变更就触发 Reconcile 方法的执行
		Owns(&networkingV1.Ingress{}).
		Named("mydeployment").
		Complete(r)
}

func (r *MyDeploymentReconciler) createDeployment(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	deployment := NewDeployment(myDeployment)

	// 设置 Deployment 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &deployment, r.Scheme)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, &deployment)
}

func (r *MyDeploymentReconciler) updateDeployment(ctx context.Context, myDeployment *myApiV1.MyDeployment, prev *appsV1.Deployment) error {
	deployment := NewDeployment(myDeployment)

	// 设置 Deployment 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &deployment, r.Scheme)
	if err != nil {
		return err
	}
	// 预更新，得到更新后的数据
	err = r.Update(ctx, &deployment, client.DryRunAll)
	if err != nil {
		return err
	}
	// 和之前的数据进行比较，如果相同，说明更新不需要
	if reflect.DeepEqual(deployment.Spec, prev.Spec) {
		return nil
	}

	return r.Client.Update(ctx, &deployment)
}

func (r *MyDeploymentReconciler) createService(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	service := NewService(myDeployment)
	// 设置 Service 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &service, r.Scheme)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, &service)
}

func (r *MyDeploymentReconciler) updateService(ctx context.Context, myDeployment *myApiV1.MyDeployment, prev *coreV1.Service) error {
	service := NewService(myDeployment)
	// 设置 Service 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &service, r.Scheme)
	if err != nil {
		return err
	}

	// 预更新，得到更新后的数据
	err = r.Update(ctx, &service, client.DryRunAll)
	if err != nil {
		return err
	}
	// 和之前的数据进行比较，如果相同，说明更新不需要
	if reflect.DeepEqual(service.Spec, prev.Spec) {
		return nil
	}
	return r.Client.Update(ctx, &service)
}

func (r *MyDeploymentReconciler) createIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	ingress := NewIngress(myDeployment)
	// 设置 Ingress 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &ingress, r.Scheme)
	if err != nil {
		return err
	}
	return r.Client.Create(ctx, &ingress)
}

func (r *MyDeploymentReconciler) updateIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment, prev *networkingV1.Ingress) error {
	ingress := NewIngress(myDeployment)

	// 设置 Ingress 所属于 md
	err := controllerutil.SetControllerReference(myDeployment, &ingress, r.Scheme)
	if err != nil {
		return err
	}
	// 预更新，得到更新后的数据
	err = r.Update(ctx, &ingress, client.DryRunAll)
	if err != nil {
		return err
	}
	// 和之前的数据进行比较，如果相同，说明更新不需要
	if reflect.DeepEqual(ingress.Spec, prev.Spec) {
		return nil
	}
	return r.Client.Update(ctx, &ingress)
}

func (r *MyDeploymentReconciler) deleteIngress(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	ingress := NewIngress(myDeployment)
	return r.Client.Delete(ctx, &ingress)
}

func (r *MyDeploymentReconciler) createIssuer(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	// 1. 创建 issuer
	issuer, err := NewIssuer(myDeployment)
	if err != nil {
		return err
	}
	// 设置 issuer 所属于 md
	err = controllerutil.SetControllerReference(myDeployment, issuer, r.Scheme)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	// 在 k8s 中创建 issuer 资源
	_, err = r.DynamicClient.Resource(issuerGVR).Namespace(myDeployment.Namespace).Create(ctx, issuer, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (r *MyDeploymentReconciler) createCertificate(ctx context.Context, myDeployment *myApiV1.MyDeployment) error {
	// 1. 创建 certificate
	certificate, err := NewCertificate(myDeployment)
	if err != nil {
		return err
	}
	// 设置 certificate 所属于 md
	err = controllerutil.SetControllerReference(myDeployment, certificate, r.Scheme)
	if err != nil {
		return err
	}
	// 在 k8s 中创建 certificate 资源
	_, err = r.DynamicClient.Resource(certificateGVR).Namespace(myDeployment.Namespace).Create(ctx, certificate, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// 更新 Condition ，并变更版本
func (r *MyDeploymentReconciler) updateConditions(myDeployment *myApiV1.MyDeployment, conditionType, message, status, reason string) {
	// 1. 获取 MyDeployment 的 status
	// 2. 获取 Conditions 字段
	// 3. 根据当前的需求，获取指定的 Condition
	var condition *myApiV1.Condition
	for i := range myDeployment.Status.Conditions {
		// 4. 是否获取到
		if myDeployment.Status.Conditions[i].Type == conditionType {
			// 4.1 获取到了
			condition = &myDeployment.Status.Conditions[i]
		}
	}
	// 4.1.1 获取当前线上的 Condition 状态，与存储的 Condition 进行比较，如果相同，跳过，不同，则进行替换
	if condition != nil {
		if condition.Status != status || condition.Reason != reason || condition.Message != message {
			condition.Status = status
			condition.Reason = reason
			condition.Message = message

		}
	} else {
		// 4.2 没获取到，创建这个 Condition，更新到 Conditions 中
		tempCondition := createCondition(conditionType, message, status, reason)
		// 追加
		myDeployment.Status.Conditions = append(myDeployment.Status.Conditions, tempCondition)
	}
	myDeployment.Status.ObservedGeneration++
}

// 判断本次reconcile是否达到预期
func (r *MyDeploymentReconciler) Ready(myDeployment *myApiV1.MyDeployment) bool {
	totalPhase, totalMessage, totalReason, success := isSuccess(myDeployment.Status.Conditions)
	// 6.1 遍历所有的 Conditions 状态，如果有任意一个 Condition 状态不是完成的状态，则将这个状态更新到总的 status 中，等待一段时间再次入队
	if !success {
		if myDeployment.Status.Phase != totalPhase || myDeployment.Status.Reason != totalReason ||
			myDeployment.Status.Message != totalMessage {
			myDeployment.Status.Phase = totalPhase
			myDeployment.Status.Reason = totalReason
			myDeployment.Status.Message = totalMessage
		}
	} else {
		// 6.2 如果所有 Conditions 的状态都为成功，则更新总的 status 为成功
		myDeployment.Status.Message = myApiV1.StatusMessageSuccess
		myDeployment.Status.Reason = myApiV1.StatusReasonSuccess
		myDeployment.Status.Phase = myApiV1.StatusPhaseComplete
		myDeployment.Status.ObservedGeneration++
	}
	// 7. 执行更新
	return success
}

func isSuccess(conditions []myApiV1.Condition) (phase string, message string, reason string, success bool) {
	if len(conditions) == 0 {
		return "", "", "", false
	}
	for i := range conditions {
		if conditions[i].Status == myApiV1.ConditionStatusFalse {
			return conditions[i].Type, conditions[i].Message, conditions[i].Reason, false
		}
	}
	return "", "", "", true
}

func createCondition(conditionType, message, status, reason string) myApiV1.Condition {
	return myApiV1.Condition{
		Type:               conditionType,
		Message:            message,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

// 需要是幂等的，可以多次执行，不管是否存在，如果存在就删除，不存在就什么也不做
// 只是删除对应的 Condition，不做更多的操作
func (r *MyDeploymentReconciler) deleteStatus(myDeployment *myApiV1.MyDeployment, conditionType string) {
	// 1. 遍历 Conditions
	var tmp []myApiV1.Condition
	copy(tmp, myDeployment.Status.Conditions)
	for i := range tmp {
		// 2. 找到要删除的对象
		if tmp[i].Type == conditionType {
			// 3. 执行删除
			myDeployment.Status.Conditions = deleteCondition(tmp, i)
		}
	}
}

func deleteCondition(conditions []myApiV1.Condition, i int) []myApiV1.Condition {
	// 前提：切片中的元素顺序不敏感

	// 1. 要删除的元素的索引值不能大于切片长度
	if i >= len(conditions) {
		return conditions
	}
	// 2. 如果切片长度为1，且索引值为0，直接清空
	if len(conditions) == 1 && i == 0 {
		return conditions[:0]
	}

	// 3. 如果长度 -1 等于索引值，删除最后一个元素
	if len(conditions)-1 == i {
		return conditions[:len(conditions)-1]

	}

	// 4. 交换索引位置的元素和最后一位，删除最后一个元素
	conditions[i], conditions[len(conditions)-1] = conditions[len(conditions)-1], conditions[i]
	conditions = conditions[:len(conditions)-1]
	return conditions
}
