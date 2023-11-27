package template

import "bytes"

func FromCmdInit(name, pkgs, dbUpdate, initDb, initCache, dbUpdateRedis string, buffer *bytes.Buffer) {
	buffer.WriteString(`
package cmd

import (
	`)
	buffer.WriteString(pkgs)
	buffer.WriteString(`
	"`)
	buffer.WriteString(name)
	buffer.WriteString(`/conf"

	"github.com/sunmi-OS/gocore/v2/conf/nacos"
	"github.com/sunmi-OS/gocore/v2/conf/viper"
	"github.com/sunmi-OS/gocore/v2/db/orm"
	"github.com/sunmi-OS/gocore/v2/db/redis"
	"github.com/sunmi-OS/gocore/v2/glog"
	"github.com/sunmi-OS/gocore/v2/glog/zap"
	"github.com/sunmi-OS/gocore/v2/utils"
)

func initConf() {
	`)
	if goCoreConfig.Config.CNacos {
		buffer.WriteString(`
		switch utils.GetRunTime() {
		case "local":
			nacos.SetLocalConfig(conf.LocalConfig)
		default:
			nacos.NewNacosEnv()
		}

		vt := nacos.GetViper()
		vt.SetBaseConfig(conf.BaseConfig)
		vt.SetDataIds(conf.ProjectName, "config" `)
		if len(goCoreConfig.Config.CMysql) > 0 {
			buffer.WriteString(`, "mysql" `)
		}
		if len(goCoreConfig.Config.CRedis) > 0 {
			buffer.WriteString(`, "redis"`)
		}
		if goCoreConfig.Config.CRocketMQConfig {
			buffer.WriteString(`, "rocketmq"`)
		}
		buffer.WriteString(`)
		// 注册配置更新回调
		vt.SetCallBackFunc(conf.ProjectName, "config", func(namespace, group, dataId, data string) {
        	initLog()
        })

		`)
		if len(goCoreConfig.Config.CMysql) > 0 {
			buffer.WriteString(`
		vt.SetCallBackFunc(conf.ProjectName, "mysql", func(namespace, group, dataId, data string) {
			`)
			buffer.WriteString(dbUpdate)
			buffer.WriteString(`
		})
		`)
		}
		if len(goCoreConfig.Config.CRedis) > 0 {
			buffer.WriteString(`
		vt.SetCallBackFunc(conf.ProjectName, "redis", func(namespace, group, dataId, data string) {
			`)
			buffer.WriteString(dbUpdateRedis)
			buffer.WriteString(`
		})
		`)
		}
		buffer.WriteString(`
		vt.NacosToViper()
	`)
	} else {
		buffer.WriteString(`
		viper.MergeConfigToToml(conf.BaseConfig)
	switch utils.GetRunTime() {
	case "dev":
		viper.MergeConfigToToml(conf.DevConfig)
	case "test":
		viper.MergeConfigToToml(conf.TestConfig)
	case "uat":
		viper.MergeConfigToToml(conf.UatConfig)
	case "onl":
		viper.MergeConfigToToml(conf.OnlConfig)
	default:
		viper.MergeConfigToToml(conf.LocalConfig)
	}
	`)
	}
	buffer.WriteString(`
}

// initDB 初始化DB服务 （内部方法）
func initDB() {
	`)
	buffer.WriteString(initDb)
	buffer.WriteString(`
}

// initCache 初始化redis服务 （内部方法）
func initCache() {
	`)
	buffer.WriteString(initCache)
	buffer.WriteString(`
}

// initLog init log
func initLog() {
	zap.SetLogLevel(viper.GetEnvConfig("base.logLevel").String())
}`)

}
