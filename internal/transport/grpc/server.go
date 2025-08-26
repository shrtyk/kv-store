package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/shrtyk/kv-store/pkg/cfg"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
	kv_store_v1 "github.com/shrtyk/kv-store/proto/grpc/gen"
)

type Server struct {
	wg       *sync.WaitGroup
	cfg      *cfg.GRPCCfg
	stCfg    *cfg.StoreCfg
	store    store.Store
	tl       tlog.TransactionsLogger
	metrics  metrics.Metrics
	logger   *slog.Logger
	grpcServ *grpc.Server

	kv_store_v1.UnimplementedKVStoreServer
}

func NewGRPCServer(
	wg *sync.WaitGroup,
	cfg *cfg.GRPCCfg,
	stCfg *cfg.StoreCfg,
	store store.Store,
	tl tlog.TransactionsLogger,
	metrics metrics.Metrics,
	logger *slog.Logger,
) *Server {
	s := &Server{
		wg:       wg,
		cfg:      cfg,
		stCfg:    stCfg,
		store:    store,
		tl:       tl,
		metrics:  metrics,
		logger:   logger,
		grpcServ: grpc.NewServer(),
	}

	kv_store_v1.RegisterKVStoreServer(s.grpcServ, s)
	reflection.Register(s.grpcServ)

	return s
}

func (s *Server) MustStart() {
	l, err := net.Listen("tcp", ":"+s.cfg.Port)
	if err != nil {
		msg := fmt.Sprintf("failed create net.Listener: %s", err)
		panic(msg)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.grpcServ.Serve(l); err != nil {
			msg := fmt.Sprintf("failed to start grpc server: %s", err)
			panic(msg)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.grpcServ.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.logger.Warn("grpcs server graceful shutdown time out; forcing stop")
		s.grpcServ.Stop()
		return ctx.Err()
	case <-done:
		s.logger.Info("grpc server graceful shutdown complete")
		return nil
	}
}
