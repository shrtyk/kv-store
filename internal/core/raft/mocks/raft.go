package mocks

import (
	"context"

	"github.com/shrtyk/kv-store/internal/core/ports/store"
	fsm_v1 "github.com/shrtyk/kv-store/proto/fsm/gen"
	raftapi "github.com/shrtyk/raft-core/api"
	"google.golang.org/protobuf/proto"
)

var _ raftapi.Raft = (*StubRaft)(nil)

// StubRaft is a stub implementation of raftapi.Raft for testing.
type StubRaft struct {
	store         store.Store
	errCh         chan error
	isLeader      bool
	leaderID      int
	killed        bool
	term          int64
	readOnlyData  []byte
	readOnlyError error
}

func NewStubRaft(store store.Store, isLeader bool, leaderID int) *StubRaft {
	return &StubRaft{
		store:    store,
		errCh:    make(chan error, 1),
		isLeader: isLeader,
		leaderID: leaderID,
		term:     1, // dummy term
	}
}

func (m *StubRaft) Submit(data []byte) *raftapi.SubmitResult {
	if !m.isLeader {
		return &raftapi.SubmitResult{
			IsLeader: false,
			LeaderID: m.leaderID,
		}
	}

	cmd := &fsm_v1.Command{}
	if err := proto.Unmarshal(data, cmd); err != nil {
		m.errCh <- err
		return &raftapi.SubmitResult{IsLeader: true}
	}

	switch c := cmd.Command.(type) {
	case *fsm_v1.Command_Put:
		_ = m.store.Put(c.Put.Key, c.Put.Value)
	case *fsm_v1.Command_Delete:
		_ = m.store.Delete(c.Delete.Key)
	}

	return &raftapi.SubmitResult{
		IsLeader: true,
		LogIndex: 1,
	}
}

func (m *StubRaft) ReadOnly(ctx context.Context, query []byte) (*raftapi.ReadOnlyResult, error) {
	if !m.isLeader {
		return &raftapi.ReadOnlyResult{
			IsLeader: false,
			LeaderId: m.leaderID,
		}, nil
	}

	if m.readOnlyError != nil {
		return nil, m.readOnlyError
	}

	if m.readOnlyData == nil {
		key := string(query)
		val, err := m.store.Get(key)
		if err != nil {
			return nil, err
		}
		return &raftapi.ReadOnlyResult{
			IsLeader: true,
			Data:     []byte(val),
		}, nil
	}

	return &raftapi.ReadOnlyResult{
		IsLeader: true,
		Data:     m.readOnlyData,
	}, nil
}

func (m *StubRaft) State() (int64, bool) {
	return m.term, m.isLeader
}

func (m *StubRaft) Snapshot(index int64, snapshot []byte) error {
	return nil
}

func (m *StubRaft) PersistedStateSize() (int, error) {
	return 0, nil
}

func (m *StubRaft) Start() error {
	m.killed = false
	return nil
}

func (m *StubRaft) Stop() error {
	m.killed = true
	return nil
}

func (m *StubRaft) Killed() bool {
	return m.killed
}

func (m *StubRaft) Errors() <-chan error {
	return m.errCh
}

func (m *StubRaft) SetLeader(isLeader bool) {
	m.isLeader = isLeader
}

func (m *StubRaft) SetReadOnlyResult(data []byte, err error) {
	m.readOnlyData = data
	m.readOnlyError = err
}
