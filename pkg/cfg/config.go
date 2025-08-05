package cfg

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

var path string

func init() {
	flag.StringVar(&path, "cfg_path", "", "Path to config file")

	_ = godotenv.Load()
}

type AppConfig struct {
	Env       string       `yaml:"env" env:"ENV" env-default:"production"`
	Store     StoreCfg     `yaml:"store"`
	Wal       WalCfg       `yaml:"transactional_logger"`
	Snapshots SnapshotsCfg `yaml:"snapshots"`
	HttpCfg   HttpCfg      `yaml:"http_cfg"`
}

type StoreCfg struct {
	MaxKeySize  int `yaml:"max_key" env:"MAX_KEY_SIZE_BYTES" env-default:"1024"`
	MaxValSize  int `yaml:"max_val" env:"MAX_VAL_SIZE_BYTES" env-default:"1024"`
	ShardsCount int `yaml:"shards_count" env:"SHARDS_COUNT" env-default:"64"`
}

type WalCfg struct {
	LogFileName        string        `yaml:"log_file_name" env:"LOG_NAME" env-default:"wal.log"`
	MaxSizeBytes       int64         `yaml:"log_size_bytes" env:"MAX_LOG_SIZE_BYTES" env-default:"10485760"`
	FsyncIn            time.Duration `yaml:"fsync_in" env:"FSYNC_LOG_IN" env-default:"300ms"`
	FsyncRetriesAmount int           `yaml:"http_retries" env:"FSYNC_RETRIES_ON_SHUTDOWN" env-default:"3"`
	FsyncRetryIn       time.Duration `yaml:"fsync_retry_in" env:"FSYNC_RETRY_IN" env-default:"500ms"`
}

type SnapshotsCfg struct {
	SnapshotsDir       string `yaml:"snapshots_dir" env:"SNAPSHOTS_DIR_PATH" env-default:"./data/snapshots/"`
	MaxSnapshotsAmount int    `yaml:"max_snapshots" env:"MAX_SNAPSHOTS_AMOUNT" env-default:"2"`
}

type HttpCfg struct {
	Host               string        `yaml:"host" env:"HOST" env-default:"localhost"`
	Port               string        `yaml:"port" env:"PORT" env-default:"16700"`
	ServerIdleTimeout  time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT" env-default:"5s"`
	ServerWriteTimeout time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s"`
	ServerReadTimeout  time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"10s"`
}

func ReadConfig() *AppConfig {
	cfgPath := cfgPath()

	cfg := new(AppConfig)
	if err := cleanenv.ReadConfig(cfgPath, cfg); err != nil {
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			msg := fmt.Sprintf(
				"couldn't read config data from config file or enviroment variables: %s: %s", err, envErr)
			panic(msg)
		}
	}
	return cfg
}

func cfgPath() string {
	if !flag.Parsed() {
		flag.Parse()
	}

	if path == "" {
		return os.Getenv("CONFIG_PATH")
	}

	return path
}
