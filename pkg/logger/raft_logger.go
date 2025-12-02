package logger

import (
	raftcorelog "github.com/shrtyk/raft-core/pkg/logger"
)

func RaftLoggerCfg(env string) raftcorelog.Enviroment {
	switch env {
	case envDev:
		return raftcorelog.Dev
	case envProd:
		return raftcorelog.Prod
	default:
		return raftcorelog.Prod
	}
}
