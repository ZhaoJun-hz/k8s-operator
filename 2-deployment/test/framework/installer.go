package framework

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Install struct {
	Name       string   `json:"name"`
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Path       string   `json:"path"`
	IgnoreFail bool     `json:"ignoreFail"`
	Config     *Config  `json:"-"`
}

func (i *Install) validate(root *field.Path) error {
	errs := field.ErrorList{}
	// 1. 验证 name 字段
	if strings.TrimSpace(i.Name) == "" {
		errs = append(errs, field.Invalid(root.Child("name"), i.Name, "name is required"))
	}
	// 2. 验证 cmd 字段
	if strings.TrimSpace(i.Cmd) == "" {
		errs = append(errs, field.Invalid(root.Child("cmd"), i.Cmd, "cmd is required"))
	}
	// 3. 检查并设置 path 默认值
	if strings.TrimSpace(i.Path) == "" {
		i.Path = "."
	}
	return errs.ToAggregate()
}

func (i *Install) install() error {
	// 1. 获取当前路径
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	absPath := i.Path
	if !filepath.IsAbs(absPath) {
		if absPath, err = filepath.Abs(absPath); err != nil {
			return err
		}
	}
	// 2. 切换路径
	// 2.1 判断路径是否相同
	if currentDir != absPath {
		// 2.1.1 不同，则切换路径
		if err := os.Chdir(absPath); err != nil {
			return err
		}
	}
	// 3. defer 退出前回到之前路径
	defer func() {
		_ = os.Chdir(currentDir)
	}()
	// 4. 执行命令
	cmd := exec.Command(i.Cmd, i.Args...)
	cmd.Stderr = i.Config.Stderr
	cmd.Stdout = i.Config.Stdout
	return cmd.Run()
}

// 使用步骤
// 1. installer := NewInstaller(config)
// 2. installer.Install()
type Installer struct {
	Steps []Install `json:"steps"`

	config *Config
	once   sync.Once
}

func NewInstaller(config *Config) *Installer {
	return &Installer{
		config: config,
	}
}

// 加载配置到 installer 对象
func (i *Installer) init() error {
	var error error
	// 只能执行一次
	i.once.Do(func() {
		// 安全检查，config 不能为空
		if i.config == nil || i.config.Sub("install") == nil {
			error = field.Invalid(field.NewPath("install"), nil, "install is required")
			return
		}
		installer := new(Installer)
		if err := i.config.Sub("install").Unmarshal(&installer); err != nil {
			error = err
			return
		}
		if installer.Steps != nil {
			i.Steps = installer.Steps
		}
	})
	return error
}

func (i *Installer) Install() error {
	// 1. 初始化，加载配置
	if err := i.init(); err != nil {
		return err
	}
	// 2. 判断执行的队列是否存在
	if len(i.Steps) == 0 {
		return nil
	}
	// 3. 遍历这个队列，执行 validate
	root := field.NewPath("install")
	for index, step := range i.Steps {
		fld := root.Index(index)
		if err := step.validate(fld); err != nil {
			return err
		}
	}
	// 3. 遍历这个队列，执行 install
	for _, step := range i.Steps {
		step.Config = i.config
		if err := step.install(); err != nil {
			return err
		}
	}
	return nil
}
