package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"gorm.io/gorm"
)

var gormDB *gorm.DB

type Config struct {
	DB         *GormConfig  `json:"DB"`
	UhubUniq   UhubUniqInfo `json:"UhubUniq"`
	ServerPort string       `json:"serverport"`
}

type UhubUniqInfo struct {
	RegionName string   `json:"RegionName"`
	HostIp     []string `json:"HostIp"`
	RegionId   int      `json:"RegionId"`
}
type GormConfig struct {
	Host            string `json:"host"`
	User            string `json:"user"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	Port            string `json:"port"`
	Table           string `json:"table"`
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

func NewConfig() *Config {
	return &Config{}
}

// 解析配置文件
func (c *Config) Parse(path string) error {
	cfgData, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("not found config file:", err)
		return err

	}
	err = json.Unmarshal([]byte(cfgData), &c)
	if err != nil {

		log.Fatal("parse config file failed:", err)
		return err

	}
	return nil
}
