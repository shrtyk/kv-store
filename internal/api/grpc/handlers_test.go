package grpc

// import (
// 	"context"
// 	"errors"
// 	"io"
// 	"log/slog"
// 	"sync"
// 	"testing"

// 	"github.com/shrtyk/kv-store/internal/cfg"
// 	"github.com/shrtyk/kv-store/internal/core/ports/store"
// 	metricsmocks "github.com/shrtyk/kv-store/internal/core/ports/metrics/mocks"
// 	storemocks "github.com/shrtyk/kv-store/internal/core/ports/store/mocks"
// 	tlogmocks "github.com/shrtyk/kv-store/internal/core/ports/tlog/mocks"
// 	pb "github.com/shrtyk/kv-store/proto/grpc/gen"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"
// )

// func TestHandlers(t *testing.T) {
// 	// Common setup for all tests
// 	storeCfg := &cfg.StoreCfg{
// 		MaxKeySize: 10,
// 		MaxValSize: 10,
// 	}
// 	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

// 	t.Run("Get", func(t *testing.T) {
// 		testCases := []struct {
// 			name         string
// 			key          string
// 			mockStoreGet func(*storemocks.MockStore)
// 			mockMetrics  func(*metricsmocks.MockMetrics)
// 			expectedResp *pb.GetResp
// 			expectedErr  error
// 		}{
// 			{
// 				name: "success",
// 				key:  "test-key",
// 				mockStoreGet: func(s *storemocks.MockStore) {
// 					s.EXPECT().Get("test-key").Return("test-value", nil).Once()
// 				},
// 				mockMetrics: func(m *metricsmocks.MockMetrics) {
// 					m.EXPECT().GrpcGet("test-key", mock.AnythingOfType("float64")).Return().Once()
// 				},
// 				expectedResp: &pb.GetResp{Entry: &pb.Entry{Key: "test-key", Value: "test-value"}},
// 				expectedErr:  nil,
// 			},
// 			{
// 				name: "not found",
// 				key:  "not-found-key",
// 				mockStoreGet: func(s *storemocks.MockStore) {
// 					s.EXPECT().Get("not-found-key").Return("", store.ErrNoSuchKey).Once()
// 				},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.NotFound, store.ErrNoSuchKey.Error()),
// 			},
// 			{
// 				name: "internal error",
// 				key:  "error-key",
// 				mockStoreGet: func(s *storemocks.MockStore) {
// 					s.EXPECT().Get("error-key").Return("", errors.New("internal error")).Once()
// 				},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.Internal, "internal error"),
// 			},
// 		}

// 		for _, tc := range testCases {
// 			t.Run(tc.name, func(t *testing.T) {
// 				mockStore := storemocks.NewMockStore(t)
// 				mockMetrics := metricsmocks.NewMockMetrics(t)
// 				mockTLog := tlogmocks.NewMockTransactionsLogger(t)

// 				tc.mockStoreGet(mockStore)
// 				tc.mockMetrics(mockMetrics)

// 				server := NewGRPCServer(&sync.WaitGroup{}, &cfg.GRPCCfg{}, storeCfg, mockStore, mockTLog, mockMetrics, logger)

// 				resp, err := server.Get(context.Background(), &pb.GetReq{Key: tc.key})

// 				assert.Equal(t, tc.expectedResp, resp)
// 				if tc.expectedErr != nil {
// 					assert.EqualError(t, err, tc.expectedErr.Error())
// 				} else {
// 					assert.NoError(t, err)
// 				}
// 			})
// 		}
// 	})

// 	t.Run("Put", func(t *testing.T) {
// 		testCases := []struct {
// 			name         string
// 			key          string
// 			value        string
// 			mockStorePut func(*storemocks.MockStore)
// 			mockTLog     func(*tlogmocks.MockTransactionsLogger)
// 			mockMetrics  func(*metricsmocks.MockMetrics)
// 			expectedResp *pb.PutResp
// 			expectedErr  error
// 		}{
// 			{
// 				name:  "success",
// 				key:   "test-key",
// 				value: "test-value",
// 				mockStorePut: func(s *storemocks.MockStore) {
// 					s.EXPECT().Put("test-key", "test-value").Return(nil).Once()
// 				},
// 				mockTLog: func(tl *tlogmocks.MockTransactionsLogger) {
// 					tl.EXPECT().WritePut("test-key", "test-value").Return().Once()
// 				},
// 				mockMetrics: func(m *metricsmocks.MockMetrics) {
// 					m.EXPECT().GrpcPut("test-key", mock.AnythingOfType("float64")).Return().Once()
// 				},
// 				expectedResp: &pb.PutResp{},
// 				expectedErr:  nil,
// 			},
// 			{
// 				name:         "key too large",
// 				key:          "this-key-is-too-large",
// 				value:        "value",
// 				mockStorePut: func(s *storemocks.MockStore) {},
// 				mockTLog:     func(tl *tlogmocks.MockTransactionsLogger) {},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.InvalidArgument, store.ErrKeyTooLarge.Error()),
// 			},
// 			{
// 				name:         "value too large",
// 				key:          "key",
// 				value:        "this-value-is-too-large",
// 				mockStorePut: func(s *storemocks.MockStore) {},
// 				mockTLog:     func(tl *tlogmocks.MockTransactionsLogger) {},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.InvalidArgument, store.ErrValueTooLarge.Error()),
// 			},
// 			{
// 				name:  "internal error",
// 				key:   "error-key",
// 				value: "value",
// 				mockStorePut: func(s *storemocks.MockStore) {
// 					s.EXPECT().Put("error-key", "value").Return(errors.New("internal error")).Once()
// 				},
// 				mockTLog: func(tl *tlogmocks.MockTransactionsLogger) {
// 					tl.EXPECT().WritePut("error-key", "value").Return().Once()
// 				},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.Internal, "internal error"),
// 			},
// 		}

// 		for _, tc := range testCases {
// 			t.Run(tc.name, func(t *testing.T) {
// 				mockStore := storemocks.NewMockStore(t)
// 				mockMetrics := metricsmocks.NewMockMetrics(t)
// 				mockTLog := tlogmocks.NewMockTransactionsLogger(t)

// 				tc.mockStorePut(mockStore)
// 				tc.mockTLog(mockTLog)
// 				tc.mockMetrics(mockMetrics)

// 				server := NewGRPCServer(&sync.WaitGroup{}, &cfg.GRPCCfg{}, storeCfg, mockStore, mockTLog, mockMetrics, logger)

// 				resp, err := server.Put(context.Background(), &pb.PutReq{Key: tc.key, Value: tc.value})

// 				assert.Equal(t, tc.expectedResp, resp)
// 				if tc.expectedErr != nil {
// 					assert.EqualError(t, err, tc.expectedErr.Error())
// 				} else {
// 					assert.NoError(t, err)
// 				}
// 			})
// 		}
// 	})

// 	t.Run("Delete", func(t *testing.T) {
// 		testCases := []struct {
// 			name            string
// 			key             string
// 			mockStoreDelete func(*storemocks.MockStore)
// 			mockTLog        func(*tlogmocks.MockTransactionsLogger)
// 			mockMetrics     func(*metricsmocks.MockMetrics)
// 			expectedResp    *pb.DeleteResp
// 			expectedErr     error
// 		}{
// 			{
// 				name: "success",
// 				key:  "test-key",
// 				mockStoreDelete: func(s *storemocks.MockStore) {
// 					s.EXPECT().Delete("test-key").Return(nil).Once()
// 				},
// 				mockTLog: func(tl *tlogmocks.MockTransactionsLogger) {
// 					tl.EXPECT().WriteDelete("test-key").Return().Once()
// 				},
// 				mockMetrics: func(m *metricsmocks.MockMetrics) {
// 					m.EXPECT().GrpcDelete("test-key", mock.AnythingOfType("float64")).Return().Once()
// 				},
// 				expectedResp: &pb.DeleteResp{},
// 				expectedErr:  nil,
// 			},
// 			{
// 				name: "internal error",
// 				key:  "error-key",
// 				mockStoreDelete: func(s *storemocks.MockStore) {
// 					s.EXPECT().Delete("error-key").Return(errors.New("internal error")).Once()
// 				},
// 				mockTLog: func(tl *tlogmocks.MockTransactionsLogger) {
// 					tl.EXPECT().WriteDelete("error-key").Return().Once()
// 				},
// 				mockMetrics:  func(m *metricsmocks.MockMetrics) {},
// 				expectedResp: nil,
// 				expectedErr:  status.Error(codes.Internal, "internal error"),
// 			},
// 		}

// 		for _, tc := range testCases {
// 			t.Run(tc.name, func(t *testing.T) {
// 				mockStore := storemocks.NewMockStore(t)
// 				mockMetrics := metricsmocks.NewMockMetrics(t)
// 				mockTLog := tlogmocks.NewMockTransactionsLogger(t)

// 				tc.mockStoreDelete(mockStore)
// 				tc.mockTLog(mockTLog)
// 				tc.mockMetrics(mockMetrics)

// 				server := NewGRPCServer(&sync.WaitGroup{}, &cfg.GRPCCfg{}, storeCfg, mockStore, mockTLog, mockMetrics, logger)

// 				resp, err := server.Delete(context.Background(), &pb.DeleteReq{Key: tc.key})

// 				assert.Equal(t, tc.expectedResp, resp)
// 				if tc.expectedErr != nil {
// 					assert.EqualError(t, err, tc.expectedErr.Error())
// 				} else {
// 					assert.NoError(t, err)
// 				}
// 			})
// 		}
// 	})
// }
