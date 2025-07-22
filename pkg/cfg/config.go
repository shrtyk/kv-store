package cfg

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	Env   string   `yaml:"env"`
	Store StoreCfg `yaml:"store_cfg"`
}

type StoreCfg struct {
	MaxKeySize uint16 `yaml:"max_key"`
	MaxValKey  uint16 `yaml:"max_val"`
}

func ReadConfig() *AppConfig {
	cfgPath := cfgPath()
	if cfgPath == "" {
		panic("config path empty")
	}

	_, err := os.Stat(cfgPath)
	if os.IsNotExist(err) {
		panic("config file does not exist: " + cfgPath)
	}

	cfg := new(AppConfig)
	if err = cleanenv.ReadConfig(cfgPath, cfg); err != nil {
		panic("couldn't read config: " + cfgPath)
	}
	return cfg
}

func cfgPath() string {
	var path string
	flag.StringVar(&path, "cfg_path", "", "Path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}
