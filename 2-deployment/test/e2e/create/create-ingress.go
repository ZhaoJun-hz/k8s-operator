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

// 真正的测试，测试创建 ingress 模式
func CreateIngressMyDeployment(ctx *framework.TestContext, f *framework.Framework) {

	var (
		// 1. 准备测试数据
		crFilePath    = "create/testdata/create-ingress.yaml"
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

	ginkgo.Context("create mode ingress mydeployment", func() {
		ginkgo.It("should be create mode ingress success", func() {
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
		ginkgo.It("should be exist ingress", func() {
			_, err := clientSet.NetworkingV1().Ingresses("deployment-system").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
		})
	})

	ginkgo.Context("delete mode ingress mydeployment", func() {
		ginkgo.It("should be delete mode ingress success", func() {
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

func CreateIngressMyDeploymentDefaultValue(ctx *framework.TestContext, f *framework.Framework) {
	var (
		// 1. 准备测试数据
		crNoReplicasFilePath    = "create/testdata/create-ingress-default-no-replicas.yaml"
		noReplicasObj           = &unstructured.Unstructured{Object: make(map[string]interface{})}
		crNoServiceportFilePath = "create/testdata/create-ingress-default-no-serviceport.yaml"
		noServiceportObj        = &unstructured.Unstructured{Object: make(map[string]interface{})}
		dynamicClient           dynamic.Interface
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
		err = f.LoadYamlToUnstructured(crNoReplicasFilePath, noReplicasObj)
		gomega.Expect(err).Should(gomega.BeNil())
		err = f.LoadYamlToUnstructured(crNoServiceportFilePath, noServiceportObj)
		gomega.Expect(err).Should(gomega.BeNil())
		// 4. 初始化测试用到的全局变量
		dynamicClient = ctx.CreateDynamicClient()
	})

	ginkgo.Context("create mode ingress mydeployment, but on replicas", func() {
		ginkgo.It("should be create mode ingress but on replicas success", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), noReplicasObj, metav1.CreateOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait creating done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should be exist mydeployment, and replicas eq 1", func() {
			md, err := dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), noReplicasObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			data := md.UnstructuredContent()
			replicas, ok := data["spec"].(map[string]interface{})["replicas"].(int64)
			gomega.Expect(ok).Should(gomega.BeTrue())
			gomega.Expect(int(replicas)).Should(gomega.Equal(1))
		})

		ginkgo.It("should be delete mode ingress but on replicas success", func() {
			err := dynamicClient.Resource(myGVR).Namespace("default").Delete(context.TODO(), noReplicasObj.GetName(), metav1.DeleteOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait deleting done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should not be exist mydeployment", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), noReplicasObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

	ginkgo.Context("create mode ingress mydeployment, but on serviceport", func() {
		ginkgo.It("should be create mode ingress but on serviceport success", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), noServiceportObj, metav1.CreateOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait creating done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should be exist mydeployment, and have a default serviceport", func() {
			md, err := dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), noServiceportObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			data := md.UnstructuredContent()
			port, ok := data["spec"].(map[string]interface{})["port"].(int64)
			gomega.Expect(ok).Should(gomega.BeTrue())
			servicePort, ok := data["spec"].(map[string]interface{})["expose"].(map[string]interface{})["servicePort"].(int64)
			gomega.Expect(ok).Should(gomega.BeTrue())
			gomega.Expect(int(servicePort)).Should(gomega.Equal(int(port)))
		})

		ginkgo.It("should be delete mode ingress but on serviceport success", func() {
			err := dynamicClient.Resource(myGVR).Namespace("default").Delete(context.TODO(), noServiceportObj.GetName(), metav1.DeleteOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait deleting done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should not be exist mydeployment", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), noServiceportObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

}

func CreateIngressMyDeploymentMustFailed(ctx *framework.TestContext, f *framework.Framework) {
	var (
		// 1. 准备测试数据
		crFilePath    = "create/testdata/create-ingress-error-no-domain.yaml"
		obj           = &unstructured.Unstructured{Object: make(map[string]interface{})}
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
		err = f.LoadYamlToUnstructured(crFilePath, obj)
		gomega.Expect(err).Should(gomega.BeNil())
		// 4. 初始化测试用到的全局变量
		dynamicClient = ctx.CreateDynamicClient()
	})

	ginkgo.Context("create mode ingress mydeployment, but error no domain", func() {
		ginkgo.It("should be create mode ingress but on replicas failed", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})
}
