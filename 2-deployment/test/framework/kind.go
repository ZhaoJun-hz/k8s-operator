package framework

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
)

type KindConfig struct {
	// 创建 cluster 的参数使用
	Name string `json:"name"`
	// 这个 config 会当作 kind create cluster --config 中的内容传入
	Config string `json:"config"`
	// 在执行完测试任务后，是否保留
	Retain bool `json:"retain"`
}

type KindProvider struct {
}

// 检查这个对象是否实现了接口ClusterProvider，这种方法也推荐在一切实现了某个接口的对象下来检查
var _ ClusterProvider = &KindProvider{}

func (KindProvider *KindProvider) Validate(config *Config) error {
	// 1. 获取配置
	if config == nil {
		return field.Invalid(field.NewPath("config"), nil, "config is required")
	}
	kindConfig := config.Sub("cluster").Sub("kind")
	root := field.NewPath("cluster", "kind")
	if kindConfig == nil {
		return field.Invalid(root, nil, "kind config is required")
	}
	// 2. 检查必要项
	if kindConfig.GetString("name") == "" {
		// 3. 设置默认项
		kindConfig.Set("name", "e2e")
	}

	return nil
}
func (KindProvider *KindProvider) Deploy(config *Config) (ClusterConfig, error) {
	clusterConfig := ClusterConfig{}
	// 1. 获取配置
	kindConfig, err := getKindConfig(config.Sub("cluster").Sub("kind"))
	if err != nil {
		return clusterConfig, err
	}
	var kubeConfigFile string
	// 2. 确认是否存在cluster
	if kindConfig.Name != "" {
		// 判断集群是否存在的命令 kind get kubeconfig --name <kindConfig.Name>
		// 用来接收配置文件的内容
		output := &bytes.Buffer{}
		cmd := exec.Command("kind", "get", "kubeconfig", "--name",
			kindConfig.Name)
		cmd.Stdout = output
		cmd.Stderr = config.Stderr
		err := cmd.Run()
		if err == nil {
			// 2.1 存在，生成访问 k8s 集群的 config 文件
			err := os.WriteFile(KubeconfigTempFile, output.Bytes(), os.ModePerm)
			if err != nil {
				return clusterConfig, err
			}
			kubeConfigFile = KubeconfigTempFile
		}
	}
	// 2.2 不存在，就创建 k8s 集群，并返回访问 k8s 集群的配置文件
	if kubeConfigFile == "" {
		// 创建集群的命令 kind create cluster --kubeconfig <KubeconfigTempFile> --config <KindConfigTempFile>
		subCommand := []string{"create", "cluster", "--kubeconfig", KubeconfigTempFile}
		if kindConfig.Config != "" {
			err := os.WriteFile(KindConfigTempFile, []byte(kindConfig.Config), os.ModePerm)
			if err != nil {
				return clusterConfig, err
			}
			defer func() {
				_ = os.Remove(KindConfigTempFile)
			}()
			subCommand = append(subCommand, "--config", KindConfigTempFile)
			subCommand = append(subCommand, "--image=my.harbor.cn/k8sstudy/kindest/node:v1.31.4")
			//subCommand = append(subCommand, "--wait", "1000")
		}
		cmd := exec.Command("kind", subCommand...)
		cmd.Stdout = config.Stdout
		cmd.Stderr = config.Stderr
		s := cmd.String()
		fmt.Println(s)
		createClusterErr := cmd.Run()
		if createClusterErr != nil {
			return clusterConfig, createClusterErr
		}
		// 退出函数之前，清空 kubeconfig 文件内容
		defer func() {
			_ = os.Remove(KubeconfigTempFile)

		}()
	}
	// 3. 创建ClusterConfig
	clusterConfig.Name = kindConfig.Name
	clusterConfig.Rest, err = clientcmd.BuildConfigFromFlags("", KubeconfigTempFile)
	if err != nil {
		return clusterConfig, err
	}
	clusterConfig.MasterIP = clusterConfig.Rest.Host
	return clusterConfig, nil
}

func (KindProvider *KindProvider) Destroy(config *Config) error {
	// 1. 根据 config 判断是否保留集群
	// 1.1 获取配置
	kindConfig, err := getKindConfig(config.Sub("cluster").Sub("kind"))
	if err != nil {
		return err
	}
	// 2. 保留就退出
	if kindConfig.Retain {
		return nil
	}
	// 3. 不保留就销毁
	// 销毁集群的命令 kind delete cluster --name <kindConfig.Name>
	cmd := exec.Command("kind", "delete", "cluster", "--name", kindConfig.Name)
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func getKindConfig(config *viper.Viper) (KindConfig, error) {
	kindConfig := KindConfig{}
	if config == nil {
		return kindConfig, field.Invalid(field.NewPath("cluster", "kind"),
			nil, "kind config is required")
	}
	if err := config.Unmarshal(&kindConfig); err != nil {
		return kindConfig, err
	}
	if kindConfig.Name == "" {
		kindConfig.Name = "e2e"
	}
	return kindConfig, nil
}
