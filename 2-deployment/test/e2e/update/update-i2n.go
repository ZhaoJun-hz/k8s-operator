package update

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

// 真正的测试函数，测试从 ingress 模式更新为 nodePort 模式
func UpdateI2NMyDeployment(ctx *framework.TestContext, f *framework.Framework) {
	var (
		// 1. 准备测试数据
		crFilePath       = "update/testdata/update-ingress.yaml"
		crUpdateFilePath = "update/testdata/update-i2n.yaml"
		obj              = &unstructured.Unstructured{Object: make(map[string]interface{})}
		updateObj        = &unstructured.Unstructured{Object: make(map[string]interface{})}

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
		err = f.LoadYamlToUnstructured(crUpdateFilePath, updateObj)
		gomega.Expect(err).Should(gomega.BeNil())
		// 4. 初始化测试用到的全局变量
		dynamicClient = ctx.CreateDynamicClient()
		clientSet = ctx.CreateClientSet()
	})
	ginkgo.Context("update mode ingress to mode nodeport mydeployment", func() {
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

		ginkgo.It("should be update to mode nodeport success", func() {
			myDeployment, err := dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.BeNil())

			updateObj.SetResourceVersion(myDeployment.GetResourceVersion())
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Update(context.TODO(), updateObj, metav1.UpdateOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait deleting done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should not be exist ingress", func() {
			_, err := clientSet.NetworkingV1().Ingresses("deployment-system").Get(context.TODO(), updateObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})

	ginkgo.Context("delete mode i2n mydeployment", func() {
		ginkgo.It("should be delete mode i2n success", func() {
			err := dynamicClient.Resource(myGVR).Namespace("default").Delete(context.TODO(), updateObj.GetName(), metav1.DeleteOptions{})
			gomega.Expect(err).Should(gomega.BeNil())
			ginkgo.By("sleep 10 second wait deleting done")
			time.Sleep(10 * time.Second)
		})
		ginkgo.It("should not be exist mydeployment", func() {
			_, err = dynamicClient.Resource(myGVR).Namespace("default").Get(context.TODO(), updateObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist deployment", func() {
			_, err := clientSet.AppsV1().Deployments("deployment-system").Get(context.TODO(), updateObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist service", func() {
			_, err := clientSet.CoreV1().Services("deployment-system").Get(context.TODO(), updateObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
		ginkgo.It("should not be exist ingress", func() {
			_, err := clientSet.NetworkingV1().Ingresses("deployment-system").Get(context.TODO(), updateObj.GetName(), metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.BeNil())
		})
	})
}
