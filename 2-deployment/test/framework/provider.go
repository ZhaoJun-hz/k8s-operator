package framework

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type ClusterProvider interface {
	Validate(config *Config) error
	Deploy(config *Config) (ClusterConfig, error)
	Destroy(config *Config) error
}

// 1. 定义工厂对象
type Factory struct {
}

// 2. 工厂对象中创建不同实现的对象
func (f Factory) Provider(config *Config) (ClusterProvider, error) {
	var clusterProvider ClusterProvider
	// 1. 检查配置
	if config.Viper == nil {
		return clusterProvider, field.Invalid(field.NewPath("config"), nil, "config is required")

	}
	// 2. 检查创建集群相关的 config
	if config.Sub("cluster") == nil {
		return clusterProvider, field.Invalid(field.NewPath("cluster"), nil, "cluster is required")
	}
	cluster := config.Sub("cluster")

	// 3. 判断创建 k8s 集群的插件，调用这个插件来创建对象
	switch {
	case cluster.Sub("kind") != nil:
		kind := new(KindProvider)
		return kind, nil
	default:
		return clusterProvider, fmt.Errorf("not support provider: %#v", cluster.AllSettings())
	}
}
