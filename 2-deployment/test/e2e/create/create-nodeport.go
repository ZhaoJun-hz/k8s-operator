package create

import (
	"context"
	"deployment/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"time"
)

// 真正的测试，测试创建 nodeport 模式
func CreateNodePortMyDeployment(ctx *framework.TestContext, f *framework.Framework) {
	var (
		// 1. 准备测试数据
		crFilePath    = "create/testdata/create-nodeport.yaml"
		obj           = &unstructured.Unstructured{Object: make(map[string]interface{})}
		dynamicClient dynamic.Interface
		clientSet     *kubernetes.Clientset
		// 3. 准备测试用到的全局变量
		myGVR = schema.GroupVersionResource{
			Group:    "apps.shudong.com",
			Version:  "v1",
			Resource: "mydeployments",
		}
		err error
	)

	ginkgo.BeforeEach(func() {
		// 2. 加载测试数据
		err = f.LoadYamlToUnstructured(crFilePath, obj)
		gomega.Expect(err).Should(gomega.BeNil())
		// 4. 初始化测试用到的全局变量
		dynamicClient = ctx.CreateDynamicClient()
		clientSet = ctx.CreateClientSet()
	})
	ginkgo.Context("create mode nodeport mydeployment", func() {
		ginkgo.It("should be create mode nodeport success", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait creating done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should be exist mydeployment", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
		})
		ginkgo.It("should be exist deployment", func() {
			_, err := clientSet.AppsV1().Deployments("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
		})
		ginkgo.It("should be exist service", func() {
			_, err := clientSet.CoreV1().Services("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
		})
		ginkgo.It("should not be exist ingress", func() {
			_, err := clientSet.NetworkingV1().Ingresses("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

	ginkgo.Context("delete mode nodeport mydeployment", func() {
		ginkgo.It("should be delete mode nodeport success", func() {
			err := dynamicClient.Resource(myGVR).Namespace("default").Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait deleting done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should not be exist mydeployment", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist deployment", func() {
			_, err := clientSet.AppsV1().Deployments("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist service", func() {
			_, err := clientSet.CoreV1().Services("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist ingress", func() {
			_, err := clientSet.NetworkingV1().Ingresses("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})
}

func CreateNodePortMyDeploymentMustFailed(ctx *framework.TestContext, f *framework.Framework) {
	var (
		// 1. 准备测试数据
		crGt32767FilePath    = "create/testdata/create-nodeport-error-gt-32767.yaml"
		gt32767obj           = &unstructured.Unstructured{Object: make(map[string]interface{})}
		crLt30000FilePath    = "create/testdata/create-nodeport-error-lt-30000.yaml"
		lt30000obj           = &unstructured.Unstructured{Object: make(map[string]interface{})}
		crNoNodePortFilePath = "create/testdata/create-nodeport-error-no-nodeport.yaml"
		noNodePortobj        = &unstructured.Unstructured{Object: make(map[string]interface{})}

		dynamicClient dynamic.Interface
		// 3. 准备测试用到的全局变量
		myGVR = schema.GroupVersionResource{
			Group:    "apps.shudong.com",
			Version:  "v1",
			Resource: "mydeployments",
		}
		err error
	)

	ginkgo.BeforeEach(func() {
		// 2. 加载测试数据
		err = f.LoadYamlToUnstructured(crGt32767FilePath, gt32767obj)
		gomega.Expect(err).Should(gomega.BeNil())
		err = f.LoadYamlToUnstructured(crLt30000FilePath, lt30000obj)
		gomega.Expect(err).Should(gomega.BeNil())
		err = f.LoadYamlToUnstructured(crNoNodePortFilePath, noNodePortobj)
		gomega.Expect(err).Should(gomega.BeNil())
		// 4. 初始化测试用到的全局变量
		dynamicClient = ctx.CreateDynamicClient()
	})

	ginkgo.Context("create mode nodeport mydeployment, but nodeport gt 32767", func() {
		ginkgo.It("should be create mode ingress but nodeport gt 32767 failed", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), gt32767obj, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

	ginkgo.Context("create mode nodeport mydeployment, but nodeport lt 30000", func() {
		ginkgo.It("should be create mode ingress but nodeport lt 30000 failed", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), lt30000obj, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

	ginkgo.Context("create mode nodeport mydeployment, but no nodeport", func() {
		ginkgo.It("should be create mode ingress but no nodeport", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), noNodePortobj, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})
}
