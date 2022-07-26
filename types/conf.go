package types

import "github.com/filecoin-project/go-state-types/abi"

const CONFIG = "config"

// Config Config struct
type Config struct {
	Mongodb    *Mongodb    `yaml:"mongodb" json:"mongodb"`
	Lotus      *Lotus      `yaml:"lotus" json:"lotus"`
	SyncConfig *SyncConfig `yaml:"syncConfig" json:"syncConfig"`
}

type Lotus struct {
	Token   string `yaml:"token" json:"token"`
	Address string `yaml:"address" json:"address"`
}

type Mongodb struct {
	Server   string `yaml:"server" json:"server"`
	NoAuth   bool   `yaml:"noAuth" json:"noAuth"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
}

type SyncConfig struct {
	Height     abi.ChainEpoch `yaml:"height" json:"height"`
	StopHeight abi.ChainEpoch `yaml:"stopHeight" json:"stopHeight"`
	MaxOnce    int64          `yaml:"maxOnce" json:"maxOnce"`
	TimeWait   int64          `yaml:"timeWait" json:"timeWait"`
}
