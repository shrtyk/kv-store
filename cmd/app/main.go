package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/shrtyk/kv-store/internal/cfg"
	internalRaft "github.com/shrtyk/kv-store/internal/core/raft"
	"github.com/shrtyk/kv-store/internal/core/store"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	log "github.com/shrtyk/kv-store/pkg/logger"
	raftapi "github.com/shrtyk/raft-core/api"
	"github.com/shrtyk/raft-core/raft"
	"github.com/shrtyk/raft-core/transport"
)

// @title           KV-Store API
// @version         1.0
// @description     A simple key-value store.
func main() {
	var wg sync.WaitGroup
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := cfg.ReadConfig()
	slogger := log.NewLogger(cfg.Env)

	st := store.NewStore(&wg, &cfg.Store, &cfg.ShardsCfg, slogger)
	m := pmts.NewPrometheusMetrics()

	parsedPeers, err := cfg.Raft.ParsePeers()
	if err != nil {
		slogger.Error("failed to parse peers", log.ErrorAttr(err))
		return
	}

	conns, closeConns, err := transport.SetupConnections(parsedPeers.Addrs)
	if err != nil {
		slogger.Error("failed to setup connections", log.ErrorAttr(err))
		return
	}
	defer func() {
		if err := closeConns(); err != nil {
			slogger.Error("failed to close connections", log.ErrorAttr(err))
		}
	}()

	raftCfg := cfg.Raft.MapToRaftApiCfg(cfg.Env)
	raftTransport, err := transport.NewGRPCTransport(raftCfg, conns)
	if err != nil {
		slogger.Error("failed to create raft transport", log.ErrorAttr(err))
		return
	}

	applyCh := make(chan *raftapi.ApplyMessage, 128)
	futures := internalRaft.NewApplyFuture()
	fsm := internalRaft.NewFSM(slogger, st, futures, applyCh)

	raftNode, err := raft.NewNodeBuilder(ctx, parsedPeers.Me, applyCh, fsm, raftTransport).
		WithConfig(raftCfg).
		Build()
	if err != nil {
		slogger.Error("failed to build raft node", log.ErrorAttr(err))
		return
	}

	app := NewApp()
	app.Init(
		WithCfg(cfg),
		WithStore(st),
		WithLogger(slogger),
		WithMetrics(m),
		WithRaft(raftNode),
		WithFSM(fsm),
		WithFutures(futures),
		WithRaftPublicHTTPAddrs(cfg.Raft.PublicHTTPAddrs),
	)

	app.Serve(ctx, &wg)
}
