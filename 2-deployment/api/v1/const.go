package v1

const (
	ModeIngress  = "ingress"
	ModeNodePort = "nodePort"
)

const (
	ConditionStatusTrue  = "True"
	ConditionStatusFalse = "False"

	ConditionTypeDeployment = "Deployment"
	ConditionTypeService    = "Service"
	ConditionTypeIngress    = "Ingress"

	ConditionMessageDeploymentOKFmt    = "Deployment %s is ready"
	ConditionMessageDeploymentNotOKFmt = "Deployment %s is not ready"
	ConditionMessageServiceOKFmt       = "Service %s is ready"
	ConditionMessageServiceNotOKFmt    = "Service %s is not ready"
	ConditionMessageIngressOKFmt       = "Ingress %s is ready"
	ConditionMessageIngressNotOKFmt    = "Ingress %s is not ready"

	ConditionReasonDeploymentReady    = "DeploymentReady"
	ConditionReasonDeploymentNotReady = "DeploymentNotReady"
	ConditionReasonServiceReady       = "ServiceReady"
	ConditionReasonServiceNotReady    = "ServiceNotReady"
	ConditionReasonIngressReady       = "IngressReady"
	ConditionReasonIngressNotReady    = "IngressNotReady"
)

const (
	StatusReasonSuccess  = "Success"
	StatusMessageSuccess = "Success"
	StatusPhaseComplete  = "Complete"
)
