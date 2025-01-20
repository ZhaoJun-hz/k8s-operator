//go:build e2e
// +build e2e

package e2e

import "deployment/test/e2e/update"

var _ = fmw.Describe("Update mode ingress to mode nodeport mydeployment ", update.UpdateI2NMyDeployment)

var _ = fmw.Describe("Update mode nodeport to mode ingress mydeployment", update.UpdateN2IMyDeployment)
