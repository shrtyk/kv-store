package raft

import (
	"context"
	"fmt"
	"log/slog"

	ftr "github.com/shrtyk/kv-store/internal/core/ports/futures"
	"github.com/shrtyk/kv-store/internal/core/ports/store"
	"github.com/shrtyk/kv-store/pkg/logger"
	fsm_v1 "github.com/shrtyk/kv-store/proto/fsm/gen"
	raftapi "github.com/shrtyk/raft-core/api"
	"google.golang.org/protobuf/proto"
)

var _ raftapi.FSM = (*storeFSM)(nil)

type storeFSM struct {
	futuresStore ftr.FuturesStore
	log          *slog.Logger
	store        store.Store
	appCh        <-chan *raftapi.ApplyMessage

	lastAppliedIdx int64
}

func NewFSM(
	log *slog.Logger,
	store store.Store,
	futureApplier ftr.FuturesStore,
	appCh <-chan *raftapi.ApplyMessage,
) raftapi.FSM {
	return &storeFSM{
		log:          log,
		store:        store,
		appCh:        appCh,
		futuresStore: futureApplier,
	}
}

func (f *storeFSM) Start(ctx context.Context) {
	f.log.Info("starting fsm")
	for {
		select {
		case <-ctx.Done():
			f.log.Info("fsm is shutting down")
			return
		case msg := <-f.appCh:
			if msg.CommandValid {
				f.applyCommand(msg.Command)
				f.lastAppliedIdx = msg.CommandIndex
				f.futuresStore.Fulfill(msg.CommandIndex)
			}
			if msg.SnapshotValid {
				if err := f.Restore(msg.Snapshot); err != nil {
					f.log.Error("failed to restore snapshot", logger.ErrorAttr(err))
					panic("failed to restore snapshot: " + err.Error())
				}
			}
		}
	}
}

func (f *storeFSM) applyCommand(data []byte) {
	var cmd fsm_v1.Command
	if err := proto.Unmarshal(data, &cmd); err != nil {
		f.log.Error("failed to unmarshal command", logger.ErrorAttr(err))
		return
	}

	switch c := cmd.Command.(type) {
	case *fsm_v1.Command_Put:
		f.log.Debug("applying put command", slog.String("key", c.Put.Key))
		if err := f.store.Put(c.Put.Key, c.Put.Value); err != nil {
			f.log.Error("failed to apply put command", logger.ErrorAttr(err))
		}
	case *fsm_v1.Command_Delete:
		f.log.Debug("applying delete command", slog.String("key", c.Delete.Key))
		if err := f.store.Delete(c.Delete.Key); err != nil {
			f.log.Error("failed to apply delete command", logger.ErrorAttr(err))
		}
	default:
		f.log.Error("unknown command type")
	}
}

func (f *storeFSM) Snapshot() ([]byte, int64, error) {
	snapshot := &fsm_v1.SnapshotState{
		Items: f.store.Items(),
	}
	b, err := proto.Marshal(snapshot)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	return b, f.lastAppliedIdx, nil
}

func (f *storeFSM) Restore(data []byte) error {
	s := new(fsm_v1.SnapshotState)
	if err := proto.Unmarshal(data, s); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot data: %w", err)
	}

	f.store.RestoreFromSnapshot(s.Items)
	return nil
}

func (f *storeFSM) Read(query []byte) ([]byte, error) {
	key := string(query)
	value, err := f.store.Get(key)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}
