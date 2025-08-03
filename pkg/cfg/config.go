package cfg

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	Env       string       `yaml:"env"`
	Store     StoreCfg     `yaml:"store"`
	Wal       WalCfg       `yaml:"transactional_logger"`
	Snapshots SnapshotsCfg `yaml:"snapshots"`
}

type StoreCfg struct {
	MaxKeySize  int `yaml:"max_key"`
	MaxValSize  int `yaml:"max_val"`
	ShardsCount int `yaml:"shards_count"`
}

type WalCfg struct {
	LogFileName        string        `yaml:"log_file_name"`
	MaxSizeBytes       int64         `yaml:"log_size_bytes"`
	FsyncIn            time.Duration `yaml:"fsync_in"`
	FsyncRetriesAmount int           `yaml:"fsync_retries"`
	FsyncRetryIn       time.Duration `yaml:"fsync_retry_in"`
}

type SnapshotsCfg struct {
	SnapshotsDir       string `yaml:"snapshots_dir"`
	MaxSnapshotsAmount int    `yaml:"max_snapshots"`
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
