package grpc

import (
	"context"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	metricsmocks "github.com/shrtyk/kv-store/internal/core/ports/metrics/mocks"
	"github.com/shrtyk/kv-store/internal/core/ports/store"
	storemocks "github.com/shrtyk/kv-store/internal/core/ports/store/mocks"
	rmocks "github.com/shrtyk/kv-store/internal/core/raft/mocks"
	"github.com/shrtyk/kv-store/pkg/logger"
	pb "github.com/shrtyk/kv-store/proto/grpc/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serverSetup struct {
	server      *Server
	mockStore   *storemocks.MockStore
	stubRaft    *rmocks.StubRaft
	mockFutures *futuresmocks.MockFuturesStore
	mockFuture  *futuresmocks.MockFuture
	mockMetrics *metricsmocks.MockMetrics
}

func setup(t *testing.T) serverSetup {
	var wg sync.WaitGroup
	stCfg := &cfg.StoreCfg{MaxKeySize: 10, MaxValSize: 20}
	grpcCfg := &cfg.GRPCCfg{}
	mockStore := storemocks.NewMockStore(t)
	stubRaft := rmocks.NewStubRaft(mockStore, true, 1)
	mockFutures := futuresmocks.NewMockFuturesStore(t)
	mockMetrics := metricsmocks.NewMockMetrics(t)
	mockFuture := futuresmocks.NewMockFuture(t)
	addrs := []string{"http://follower:8080", "http://leader:8080"}
	slogger := logger.NewLogger("dev")

	server := NewGRPCServer(
		&wg,
		grpcCfg,
		stCfg,
		mockStore,
		mockMetrics,
		slogger,
		stubRaft,
		mockFutures,
		addrs,
	)

	return serverSetup{server, mockStore, stubRaft, mockFutures, mockFuture, mockMetrics}
}

func TestGRPCServer_Put(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "value"

		s.mockStore.On("Put", key, value).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(nil).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()
		s.mockMetrics.On("GrpcPut", key, mock.Anything).Return().Once()

		_, err := s.server.Put(context.Background(), &pb.PutReq{Key: key, Value: value})

		assert.NoError(t, err)
		s.mockStore.AssertExpectations(t)
		s.mockMetrics.AssertExpectations(t)
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)

		_, err := s.server.Put(context.Background(), &pb.PutReq{Key: "key", Value: "value"})

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
		assert.Contains(t, st.Message(), "not a leader")
	})

	t.Run("key too large", func(t *testing.T) {
		s := setup(t)
		_, err := s.server.Put(context.Background(), &pb.PutReq{Key: "thiskeyistoolarge", Value: "value"})
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("value too large", func(t *testing.T) {
		s := setup(t)
		_, err := s.server.Put(context.Background(), &pb.PutReq{Key: "key", Value: "thisvalueistoolargetoomuch"})
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("future timeout", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "value"

		s.mockStore.On("Put", key, value).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(context.DeadlineExceeded).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()

		_, err := s.server.Put(context.Background(), &pb.PutReq{Key: key, Value: value})

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestGRPCServer_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "value"

		s.stubRaft.SetReadOnlyResult([]byte(value), nil)
		s.mockMetrics.On("GrpcGet", key, mock.Anything).Return().Once()

		resp, err := s.server.Get(context.Background(), &pb.GetReq{Key: key})

		assert.NoError(t, err)
		assert.Equal(t, key, resp.Entry.Key)
		assert.Equal(t, value, resp.Entry.Value)
		s.mockStore.AssertExpectations(t)
		s.mockMetrics.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		s := setup(t)
		key := "notfound"

		s.stubRaft.SetReadOnlyResult(nil, store.ErrNoSuchKey)

		_, err := s.server.Get(context.Background(), &pb.GetReq{Key: key})

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)

		_, err := s.server.Get(context.Background(), &pb.GetReq{Key: "key"})

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
		assert.Contains(t, st.Message(), "not a leader")
	})
}

func TestGRPCServer_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key := "key"

		s.mockStore.On("Delete", key).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(nil).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()
		s.mockMetrics.On("GrpcDelete", key, mock.Anything).Return().Once()

		_, err := s.server.Delete(context.Background(), &pb.DeleteReq{Key: key})

		assert.NoError(t, err)
		s.mockStore.AssertExpectations(t)
		s.mockMetrics.AssertExpectations(t)
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)

		_, err := s.server.Delete(context.Background(), &pb.DeleteReq{Key: "key"})

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
	})
}
