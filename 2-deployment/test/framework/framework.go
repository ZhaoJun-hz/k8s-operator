package framework

import (
	"context"
	"flag"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"golang.org/x/exp/rand"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

var DefaultStartTimeout = float64(60 * 60)
var nsRegex = regexp.MustCompile("[^a-z0-9]")

type Framework struct {
	Config        *Config
	ClusterConfig *ClusterConfig

	// 工厂对象，提供创建 provider 的方法
	factory Factory
	// 存储当前 Framework 对象中实现的 provider
	provider ClusterProvider

	// 用于连接创建的 k8s 集群
	client kubernetes.Interface
	// 配置文件启动路径
	configFile string
	// 启动时候，包括安装集群和依赖及本程序的超时时间
	initTimeout float64
}

func NewFramework() *Framework {
	return &Framework{}
}

// 解析命令行参数 --config --timeout
func (f *Framework) Flags() *Framework {
	flag.StringVar(&f.configFile, "config", "config", "Path to config file")
	flag.Float64Var(&f.initTimeout, "startup-timeout", DefaultStartTimeout, "startup timeout")
	// 让配置生效，就是将参数赋值到变量中
	flag.Parse()
	return f
}

// 加载配置文件到 framework
func (f *Framework) LoadConfig(writer io.Writer) *Framework {
	// 1. 创建 config 对象
	config := NewConfig()
	// 2. 加载配置文件内容到 config 对象中
	if err := config.Load(f.configFile); err != nil {
		panic(err)
	}
	// 3. 将传入的 writer 应用到 config 中
	config.WithWriter(writer)
	// 4. 将 config 加入到 framework 中，
	f.WithConfig(config)
	return f
}

func (f *Framework) SynchronizedBeforeSuite(initFunc func()) *Framework {
	if initFunc == nil {
		initFunc = func() {
			// 真实的before执行的内容
			// KubeconfigTempFile 是否存在

			if !exists(KubeconfigTempFile) {
				// 创建文件
				file, err := os.Create(KubeconfigTempFile)
				if err != nil {
					panic(err)
				}
				defer file.Close()
			}

			// 1. 安装环境
			ginkgo.By("Deploying test environment")
			if err := f.DeployTestEnvironment(); err != nil {
				panic(err)
			}
			// 2. 初始化环境访问的授权，也就是创建 kubectl 访问需要的 config
			ginkgo.By("kubectl switch context")
			kubectlConfig := NewKubectlConfig(f.Config)
			if err := kubectlConfig.SetContext(f.ClusterConfig); err != nil {
				panic(err)
			}
			// 退出前清理 context
			defer func() {
				ginkgo.By("kubectl reverting context")
				if !f.Config.Sub("cluster").Sub("kind").GetBool("retain") {
					_ = kubectlConfig.DeleteContext(f.ClusterConfig)
				}
			}()
			// 3. 安装依赖和自己的程序
			ginkgo.By("Preparing install steps")
			installer := NewInstaller(f.Config)
			ginkgo.By("Executing install steps")
			if err := installer.Install(); err != nil {
				panic(err)
			}
		}
	}
	ginkgo.SynchronizedBeforeSuite(func() []byte {
		initFunc()
		return nil
	}, func(_ []byte) {

	}, f.initTimeout)
	return f
}

func (f *Framework) SynchronizedAfterSuite(destroyFunc func()) *Framework {
	if destroyFunc == nil {
		destroyFunc = func() {
			// 真实的 after 执行的内容，回收测试环境
			if err := f.DestroyTestEnvironment(); err != nil {
				panic(err)
			}
		}
	}
	ginkgo.SynchronizedAfterSuite(func() {

	}, destroyFunc, f.initTimeout)
	return f
}

func (f *Framework) MRun(m *testing.M) {
	// 优化随机数
	rand.Seed(uint64(time.Now().UnixNano()))
	// 执行真正的 TestMain
	os.Exit(m.Run())
}

func (f *Framework) Run(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	// TODO
	//var r []ginkgo.Reporter
	//r = append(r, reporters.NewJUnitReporter("e2e.xml"))
	//ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "e2e", r)
	ginkgo.RunSpecs(t, "framework")
}

func (f *Framework) WithConfig(config *Config) *Framework {
	f.Config = config
	return f
}

// 创建测试环境，获取访问集群的配置及 client
func (f *Framework) DeployTestEnvironment() error {
	// 1. 检查 f.config
	if f.Config == nil {
		return field.Invalid(field.NewPath("config"), nil, "config is required")
	}

	// 2. 创建 provider
	ginkgo.By("Getting env provider")
	clusterProvider, err := f.factory.Provider(f.Config)
	if err != nil {
		return err
	}
	f.provider = clusterProvider

	// 3. 执行 provider 实现的 validate 方法验证 config
	ginkgo.By("Validating config for provider")
	err = f.provider.Validate(f.Config)
	if err != nil {
		return err
	}

	// 4. 执行 provider 实现的 Deploy 方法，创建集群
	ginkgo.By("Deploying test env")
	clusterConfig, err := f.provider.Deploy(f.Config)
	if err != nil {
		return err
	}
	f.ClusterConfig = &clusterConfig
	// 5. 创建 client 用于执行测试用例的时候使用
	client, err := kubernetes.NewForConfig(f.ClusterConfig.Rest)
	if err != nil {
		return err
	}
	f.client = client
	return nil

}

// 销毁测试环境，此方法要在执行过 DeployTestEnvironment 方法后执行
func (f *Framework) DestroyTestEnvironment() error {
	// 1. 检查 f.Config
	if f.Config == nil {
		return field.Invalid(field.NewPath("config"), nil, "config is required")
	}

	// 2. 检查 provider
	if f.provider == nil {
		return fmt.Errorf("f.provider is nil")
	}
	// 3. 执行 provider 实现的 Destroy 方法，销毁环境
	ginkgo.By("Destroy test environment")
	err := f.provider.Destroy(f.Config)
	if err != nil {
		return err
	}

	// 4. 清空 f.provider，保护销毁函数被多次执行而报错
	f.provider = nil

	return nil
}

func (f *Framework) Describe(name string, ctxFunc ContextFunc) bool {
	// 整个函数，实际上是调用 ginkgo 的 Describe
	return ginkgo.Describe(name, func() {
		// 1. 创建 testContext
		ctx, err := f.createTestContext(name, false)
		if err != nil {
			ginkgo.Fail("cannot create test context for " + name)
			return
		}
		// 2. 执行每次测试任务前，来执行一些期望的动作，如创建 namespace 就放在这里
		ginkgo.BeforeEach(func() {
			ctx2, err := f.createTestContext(name, true)
			if err != nil {
				ginkgo.Fail("cannot create test context for " + name + " namespace = " + ctx2.Namespace)
				return
			}
			ctx = ctx2
		})
		// 3. 执行每次测试任务后，来执行一些期望的动作，如删除 testContext
		ginkgo.AfterEach(func() {
			_ = f.deleteTestContext(ctx)
		})
		// 4. 执行用户的测试函数
		ctxFunc(&ctx, f)
	})
}

// 加载测试文件内容到对象中
func (f *Framework) LoadYamlToUnstructured(path string, obj *unstructured.Unstructured) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &(obj.Object))
}

func (f *Framework) createTestContext(name string, nsCreate bool) (TestContext, error) {
	// 1. 创建 TestContext 对象
	testContext := TestContext{}
	// 2. 检查 f 是否为空
	if f.Config == nil || f.ClusterConfig == nil {
		// TODO
		return testContext, nil
		//return testContext, field.Invalid(field.NewPath("config/clusterConfig"), nil, "config is required")
	}

	// 3. 填充字段
	testContext.Name = name
	testContext.Config = rest.CopyConfig(f.ClusterConfig.Rest)
	testContext.MasterIP = f.ClusterConfig.MasterIP
	// 4. 判断参数，是否创建 namespace
	if nsCreate {
		// 4.1 如果创建，使用 f.client 来创建 namespace
		// 4.1.1 处理 name ，将空格或者下划线替换为"-"
		// 4.1.2 正则检查是否有其他非法字符
		// 4.1.3
		prefix := nsRegex.ReplaceAllString(
			strings.ReplaceAll(
				strings.ReplaceAll(strings.ToLower(name), " ", "-"),
				"_", "-"), "")
		if len(prefix) > 30 {
			prefix = prefix[:30]
		}
		namespace, err := f.client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: prefix + "-",
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return testContext, nil
		}
		testContext.Namespace = namespace.GetName()
	}

	// 5. 执行其他想要做的事情，比如要创建的 sa / secret

	return testContext, nil
}

func (f *Framework) deleteTestContext(ctx TestContext) error {
	// 删除创建的资源
	// 这里只创建了 namespace，所以只删除namespace，

	errs := field.ErrorList{}
	// 删除 namespace
	if ctx.Namespace != "" {
		err := f.client.CoreV1().Namespaces().Delete(context.TODO(), ctx.Namespace, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			errs = append(errs, field.InternalError(
				field.NewPath("testcontext"), err))
		}
	}
	// 如果创建了其他更多的资源，需要实现执行删除的操作
	// 如果执行过程中出现错误，同样 添加到 errs
	return errs.ToAggregate()
}

// 检查文件或目录是否存在
func exists(path string) bool {
	_, err := os.Stat(path)
	// 如果错误是 os.ErrNotExist，表示文件或目录不存在
	return !os.IsNotExist(err)
}
