//go:build e2e
// +build e2e

package e2e

import "deployment/test/e2e/create"

var _ = fmw.Describe("Create mode ingress mydeployment ", create.CreateIngressMyDeployment)
var _ = fmw.Describe("Create mode nodeport mydeployment ", create.CreateNodePortMyDeployment)
var _ = fmw.Describe("Create mode ingress mydeployment default value", create.CreateIngressMyDeploymentDefaultValue)
var _ = fmw.Describe("Create mode ingress mydeployment must failed", create.CreateIngressMyDeploymentMustFailed)
var _ = fmw.Describe("Create mode nodeport mydeployment must failed", create.CreateNodePortMyDeploymentMustFailed)
var _ = fmw.Describe("Create mode ingress mydeployment with tls", create.CreateIngressMyDeploymentWithTls)
