package framework

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// 1. 定义一个测试的入口函数 Describe，这里接收测试的描述以及 ContextFunc
// 1.1 这里面会调用 context 创建方法来创建 context
// 1.2 这个 context 里面会执行一些期望的行为
// 2. 这个 ContextFunc 的签名符合 func(ctx *TestContext, f *Framework)
// 3. 这个 ContextFunc 的函数体就是测试函数的内容本身
// 4. 由于这个 ContextFunc 的参数中有 ctx 入参，那么在执行测试函数体的时候，就可以使用 ctx 中的内容或方法
type TestContext struct {
	Name      string
	Namespace string
	Config    *rest.Config
	MasterIP  string
}

type ContextFunc func(ctx *TestContext, f *Framework)

// 动态的client，用来访问自定义或者后安装的资源
// 如果不用动态的client，那么当访问这些资源的时候，就需要
// 1. 自己创建 rest api 的请求
// 2. 获取对应资源的 client sdk
// 创建动态的client，用来访问自定义或者后安装的资源
func (tc *TestContext) CreateDynamicClient() dynamic.Interface {
	ginkgo.By("Creating dynamic client")
	client, err := dynamic.NewForConfig(tc.Config)
	if err != nil {
		gomega.Expect(err).Should(gomega.BeNil())
	}
	return client
}

// 创建一个clientset，用来访问内置资源
func (tc *TestContext) CreateClientSet() *kubernetes.Clientset {
	ginkgo.By("Creating ClientSet client")
	client, err := kubernetes.NewForConfig(tc.Config)
	if err != nil {
		gomega.Expect(err).Should(gomega.BeNil())
	}
	return client
}
