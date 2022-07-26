package main

import (
	"io/ioutil"

	"github.com/guowenshuai/lotus_tool/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func initConfig() *types.Config {
	// 初始化启动配置
	config := &types.Config{
		Mongodb: &types.Mongodb{
			Server:   "192.168.1.157:27037",
			NoAuth:   false,
			Username: "admin",
			Password: "admin",
			Database: "filecoin",
		},
		Lotus: &types.Lotus{
			Token:   "",
			Address: "",
		},
		SyncConfig: &types.SyncConfig{
			Height:     281520, // 2020-12-01 00:00:00
			StopHeight: 0,
			MaxOnce:    10,
			TimeWait:   10,
		},
	}
	if err := loadYaml("conf.yaml", config); err != nil {
		panic(err)
	}
	return config
}

func loadYaml(file string, config *types.Config) error {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		logrus.Printf("yamlFile.Get err   #%v ", err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		logrus.Errorf("read config err: %s", err.Error())
		return err
	}
	d, _ := yaml.Marshal(config)
	logrus.Printf("%s", string(d))
	return nil
}
