package raft

import (
	"context"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/core/ports/store"
	storemocks "github.com/shrtyk/kv-store/internal/core/ports/store/mocks"
	fsm_v1 "github.com/shrtyk/kv-store/proto/fsm/gen"
	raftapi "github.com/shrtyk/raft-core/api"
	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/proto"

	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	"github.com/shrtyk/kv-store/pkg/logger"
)

type fsmSetup struct {
	fsm         *storeFSM
	mockStore   *storemocks.MockStore
	mockFutures *futuresmocks.MockFuturesStore
	appCh       chan *raftapi.ApplyMessage
}

func setup(t *testing.T) fsmSetup {
	slogger := logger.NewLogger("dev")
	mockStore := storemocks.NewMockStore(t)
	mockFutures := futuresmocks.NewMockFuturesStore(t)
	appCh := make(chan *raftapi.ApplyMessage, 1)

	fsm := NewFSM(slogger, mockStore, mockFutures, appCh).(*storeFSM)

	return fsmSetup{fsm, mockStore, mockFutures, appCh}
}

func TestFSM_Apply(t *testing.T) {
	t.Run("put command", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "value"
		logIndex := int64(123)

		putCmd := &fsm_v1.Command{
			Command: &fsm_v1.Command_Put{
				Put: &fsm_v1.PutCommand{Key: key, Value: value},
			},
		}
		cmdBytes, err := proto.Marshal(putCmd)
		assert.NoError(t, err)

		s.mockStore.On("Put", key, value).Return(nil).Once()
		s.mockFutures.On("Fulfill", logIndex).Return().Once()

		s.appCh <- &raftapi.ApplyMessage{
			CommandValid: true,
			Command:      cmdBytes,
			CommandIndex: logIndex,
		}

		go s.fsm.Start(context.Background())
		time.Sleep(10 * time.Millisecond)

		s.mockStore.AssertExpectations(t)
		s.mockFutures.AssertExpectations(t)
	})

	t.Run("delete command", func(t *testing.T) {
		s := setup(t)
		key := "key"
		logIndex := int64(456)

		delCmd := &fsm_v1.Command{
			Command: &fsm_v1.Command_Delete{
				Delete: &fsm_v1.DeleteCommand{Key: key},
			},
		}
		cmdBytes, err := proto.Marshal(delCmd)
		assert.NoError(t, err)

		s.mockStore.On("Delete", key).Return(nil).Once()
		s.mockFutures.On("Fulfill", logIndex).Return().Once()

		s.appCh <- &raftapi.ApplyMessage{
			CommandValid: true,
			Command:      cmdBytes,
			CommandIndex: logIndex,
		}

		go s.fsm.Start(context.Background())
		time.Sleep(10 * time.Millisecond)

		s.mockStore.AssertExpectations(t)
		s.mockFutures.AssertExpectations(t)
	})
}

func TestFSM_Snapshot(t *testing.T) {
	s := setup(t)
	items := map[string]string{"key1": "val1", "key2": "val2"}
	s.fsm.lastAppliedIdx = 100

	s.mockStore.On("Items").Return(items).Once()

	snapBytes, lastIndex, err := s.fsm.Snapshot()
	assert.NoError(t, err)
	assert.Equal(t, s.fsm.lastAppliedIdx, lastIndex)

	var snapshot fsm_v1.SnapshotState
	err = proto.Unmarshal(snapBytes, &snapshot)
	assert.NoError(t, err)
	assert.Equal(t, items, snapshot.Items)
}

func TestFSM_Restore(t *testing.T) {
	s := setup(t)
	items := map[string]string{"key1": "val1", "key2": "val2"}
	snapshot := &fsm_v1.SnapshotState{Items: items}
	snapBytes, err := proto.Marshal(snapshot)
	assert.NoError(t, err)

	s.mockStore.On("RestoreFromSnapshot", items).Return().Once()

	err = s.fsm.Restore(snapBytes)
	assert.NoError(t, err)

	s.mockStore.AssertExpectations(t)
}

func TestFSM_Read(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "value"

		s.mockStore.On("Get", key).Return(value, nil).Once()

		result, err := s.fsm.Read([]byte(key))

		assert.NoError(t, err)
		assert.Equal(t, []byte(value), result)
		s.mockStore.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		s := setup(t)
		key := "notfound"

		s.mockStore.On("Get", key).Return("", store.ErrNoSuchKey).Once()

		_, err := s.fsm.Read([]byte(key))

		assert.ErrorIs(t, err, store.ErrNoSuchKey)
		s.mockStore.AssertExpectations(t)
	})
}
