package cfg

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var path string

func init() {
	flag.StringVar(&path, "cfg_path", "", "Path to config file")
}

type AppConfig struct {
	Env       string       `yaml:"env" env:"ENV" env-default:"production"`
	Store     StoreCfg     `yaml:"store"`
	ShardsCfg ShardsCfg    `yaml:"shards"`
	Wal       WalCfg       `yaml:"transactional_logger"`
	Snapshots SnapshotsCfg `yaml:"snapshots"`
	HttpCfg   HttpCfg      `yaml:"http"`
	GRPCCfg   GRPCCfg      `yaml:"grpc"`
}

type StoreCfg struct {
	MaxKeySize  int `yaml:"max_key" env:"MAX_KEY_SIZE_BYTES" env-default:"1024"`
	MaxValSize  int `yaml:"max_val" env:"MAX_VAL_SIZE_BYTES" env-default:"1024"`
	ShardsCount int `yaml:"shards_count" env:"SHARDS_COUNT" env-default:"64"`
}

type ShardsCfg struct {
	CheckFreq          time.Duration `yaml:"check_frequency" env:"SHARDS_CHECK_FREQ" env-default:"30s"`
	SparseRatio        float64       `yaml:"sparse_ratio" env:"SHARDS_SPARSE_RATIO" env-default:"0.5"`
	MinOpsUntilRebuild int           `yaml:"min_operations_until_rebuild" env:"SHARDS_MIN_OPS_UNTIL_REBUILD" env-default:"2000"`
	MinDeletes         int           `yaml:"min_deletes" env:"SHARDS_MIN_DELETES" env-default:"500"`
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
	Host               string        `yaml:"host" env:"HTTP_HOST" env-default:"localhost"`
	Port               string        `yaml:"port" env:"HTTP_PORT" env-default:"16700"`
	ServerIdleTimeout  time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"5s"`
	ServerWriteTimeout time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ServerReadTimeout  time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"10s"`
}

type GRPCCfg struct {
	Port string `yaml:"port" env:"GRPC_PORT" env-default:"3000"`
}

func ReadConfig() *AppConfig {
	cfgPath := cfgPath()

	cfg := new(AppConfig)
	if err := cleanenv.ReadConfig(cfgPath, cfg); err != nil {
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			msg := fmt.Sprintf(
				"couldn't read config data from config file or environment variables: %s: %s", err, envErr)
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
