package cmd

import (
	"io/ioutil"
	"os"

	"github.com/sunmi-OS/gocore/v2/tools/gocore/file"
	"gopkg.in/yaml.v2"

	"github.com/sunmi-OS/gocore/v2/tools/gocore/conf"

	"github.com/urfave/cli/v2"
)

// CreatYaml 创建配置文件
var CreatYaml = &cli.Command{
	Name: "yaml",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Usage: "dir path",
		}},
	Usage:  "create conf [dir]",
	Action: creatYaml,
}

// creatYaml 创建配置文件
func creatYaml(c *cli.Context) error {
	_, err := InitYaml(".", conf.GetGocoreConfig())
	if err != nil {
		return err
	}
	printHint("Welcome to GoCore, Configuration file has been generated.")
	return nil
}

// InitYaml 生成Yaml配置文件
// TODO 命名和实际操作有二义性，拆开成不同独立的操作
func InitYaml(dir string, config *conf.GoCore) (*conf.GoCore, error) {
	yamlPath := "gocore.yaml"
	if dir != "" {
		yamlPath = dir + "/gocore.yaml"
	}
	if file.CheckFileIsExist(yamlPath) {
		apiFile, err := os.Open(yamlPath)
		if err == nil {
			content, err := ioutil.ReadAll(apiFile)
			if err != nil {
				panic(err)
			}
			cfg := conf.GoCore{}
			err = yaml.Unmarshal(content, &cfg)
			if err != nil {
				panic(err)
			}
			return &cfg, nil
		}
		panic(err)
	}
	var writer = file.NewWriter()
	yamlByte, err := yaml.Marshal(config)
	if err != nil {
		return config, err
	}
	writer.Add(yamlByte)
	writer.WriteToFile(yamlPath)
	return config, nil
}