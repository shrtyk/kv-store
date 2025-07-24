package cfg

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	Env         string         `yaml:"env"`
	Store       StoreCfg       `yaml:"store_cfg"`
	TransLogger TransLoggerCfg `yaml:"transactional_logger_cfg"`
}

type StoreCfg struct {
	MaxKeySize int `yaml:"max_key"`
	MaxValSize int `yaml:"max_val"`

	TryRebuildIn        time.Duration `yaml:"rebuild_in"`
	MinDeletesTrigger   int           `yaml:"min_deletes"`
	SparseRatio         float64       `yaml:"sparse_ratio"`
	MinOpsBeforeRebuild int           `yaml:"min_ops_to_rebuild"`
}

type TransLoggerCfg struct {
	LogFileName   string        `yaml:"log_file_name"`
	FsyncIn       time.Duration `yaml:"fsync_in"`
	RetriesAmount int           `yaml:"retries"`
	RetryIn       time.Duration `yaml:"retry_in"`
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
