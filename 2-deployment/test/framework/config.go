package framework

import (
	"fmt"
	"github.com/spf13/viper"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
)

var ConfigFileGroupResource = schema.GroupResource{
	Group:    "",
	Resource: "config",
}

type Config struct {
	// 动态处理配置文件的工具
	*viper.Viper

	Stdout io.Writer
	Stderr io.Writer
}

func NewConfig() *Config {
	return &Config{
		Viper:  viper.New(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// 从文件加载配置内容到 config 对象中
func (config *Config) Load(configFile string) error {
	// 1. 设置文件名
	config.SetConfigName(filepath.Base(configFile))

	// 2. 设置文件目录
	config.AddConfigPath(filepath.Dir(configFile))
	// 3. 读入文件
	err := config.ReadInConfig()
	// 4. 错误处理
	if err != nil {
		ext := filepath.Ext(configFile)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && ext != "" {
			config.SetConfigName(filepath.Base(configFile[:len(configFile)-len(ext)]))
			err = config.ReadInConfig()
		}
		if err != nil {
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
				return errors.NewNotFound(ConfigFileGroupResource,
					fmt.Sprintf("config file \"%s\" not found", configFile))
			case viper.UnsupportedConfigError:
				return errors.NewBadRequest("unsupported config file format")
			default:
				return err
			}
		}
	}
	return nil
}

func (config *Config) WithWriter(std io.Writer) *Config {
	config.Stdout = std
	config.Stderr = std
	return config
}

type ClusterConfig struct {
	// 存储名字，这个名字在使用 kind create cluster 的时候 --name 传入
	Name string
	// 连接创建的 k8s 的 client，这个 client 比较低级
	Rest *rest.Config `json:"-"`
	// 集群的 master ip，在一些需要直接和集群通讯测试的时候使用
	MasterIP string
}
