package types

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const CONFIG = "config"

// Config Config struct
type Config struct {
	Demo          *Lotus      `yaml:"demo" json:"demo"`
	Storage       *Lotus      `yaml:"storage" json:"storage"`
	Setting       *Setting    `yaml:"setting" json:"setting"`
	Prometheus    *Prometheus `json:"prometheus" yaml:"prometheus"`
	Alert         *Alert      `json:"alert" yaml:"alert"`
	RedisAddress  string      `json:"redisAddress" yaml:"redisAddress"`   // redis address
	RedisPassword string      `json:"redisPassword" yaml:"redisPassword"` // redis password
	RedisDBNumber int         `json:"redisDBNumber" yaml:"redisDBNumber"` // redis database number
	Miners        []*Miner    `yaml:"miners" json:"miners"`
}

type Miner struct {
	Miner         string `yaml:"miner" json:"miner"`
	Wallet        string `yaml:"wallet" json:"wallet"`
	Storage       *Lotus `yaml:"storage" json:"storage"`
	ProLimit      int64  `json:"proLimit" yaml:"proLimit"`
	PreLimit      int64  `json:"preLimit" yaml:"preLimit"`
	PreBreakLimit int64  `json:"preBreakLimit" yaml:"preBreakLimit"`
	OnceMax       int    `json:"onceMax" yaml:"onceMax"`
}

type Lotus struct {
	Token   string `yaml:"token" json:"token"`
	Address string `yaml:"address" json:"address"`
}

type Setting struct {
	RefreshTime        int64      `json:"refreshTime" yaml:"refreshTime"`
	RefreshBaseFee     int64      `json:"refreshBaseFee" yaml:"refreshBaseFee"`
	BaseFee            string     `json:"baseFee" yaml:"baseFee"`
	BaseFeeMax         string     `json:"baseFeeMax" yaml:"baseFeeMax"`
	BaseFeePercent     int        `json:"baseFeePercent" yaml:"baseFeePercent"`
	TimeKeep           int64      `json:"timeKeep" yaml:"timeKeep"`
	MpoolThresholdHigh int        `json:"mpoolThresholdHigh" yaml:"mpoolThresholdHigh"`
	MpoolThresholdLow  int        `json:"mpoolThresholdLow" yaml:"mpoolThresholdLow"`
	StepFee            []*StepFee `json:"stepFee" yaml:"stepFee"`
	ProLimit           int64      `json:"proLimit" yaml:"proLimit"`
	PreLimit           int64      `json:"preLimit" yaml:"preLimit"`
	PreBreakLimit      int64      `json:"preBreakLimit" yaml:"preBreakLimit"`
	LimitAdjustSeed    int64      `json:"limitAdjustSeed" yaml:"limitAdjustSeed"`
	LimitEstimateSeed  int64      `json:"limitEstimateSeed" yaml:"limitEstimateSeed"`
	LimitMaxPremium    int64      `json:"limitMaxPremium" yaml:"limitMaxPremium"`
	PremiumSeed        int64      `json:"premiumSeed" yaml:"premiumSeed"`
	OnceMax            int        `json:"onceMax" yaml:"onceMax"`
	Addresses          []string   `json:"addresses" yaml:"addresses"`
}

type Prometheus struct {
	Port string `json:"port" yaml:"port"`
}

type Alert struct {
	DingDing *Ding `yaml:"dingding" json:"dingding"`
}

type Ding struct {
	URL string `yaml:"url" json:"url"`
}

func LoadYaml(file string, config *Config) error {
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
	return nil
}

type StepFee struct {
	Hour float64 `json:"hour" yaml:"hour"`
	Fee  string  `json:"fee" yaml:"fee"`
}
