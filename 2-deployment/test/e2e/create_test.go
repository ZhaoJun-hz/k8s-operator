//go:build e2e
// +build e2e

package e2e

import "deployment/test/e2e/create"

var _ = fmw.Describe("Create mode ingress mydeployment ", create.CreateIngressMyDeployment)
var _ = fmw.Describe("Create mode ingress mydeployment ", create.CreateNodePortMyDeployment)
