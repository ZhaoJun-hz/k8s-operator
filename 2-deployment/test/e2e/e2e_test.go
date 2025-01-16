//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"

	"deployment/test/framework"
)

var fmw = framework.NewFramework()

// 执行 go test 时候，会被先执行的内容
func TestMain(m *testing.M) {
	// 解析命令行
	fmw.Flags().
		// 加载配置
		LoadConfig(ginkgo.GinkgoWriter).
		// 同步的，在执行测试任务之前执行的内容
		SynchronizedBeforeSuite(nil).
		// 同步的，在执行测试任务之后执行的内容
		SynchronizedAfterSuite(nil).
		// 运行测试主函数
		MRun(m)
}

// 执行 go test 时候，会被后执行的内容，也就是正常的测试用例
func TestE2E(t *testing.T) {
	fmw.Run(t)
}
