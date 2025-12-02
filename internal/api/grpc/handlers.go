package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/shrtyk/kv-store/internal/core/ports/store"
	fsm_v1 "github.com/shrtyk/kv-store/proto/fsm/gen"
	pb "github.com/shrtyk/kv-store/proto/grpc/gen"
	raftapi "github.com/shrtyk/raft-core/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) Get(ctx context.Context, in *pb.GetReq) (*pb.GetResp, error) {
	start := time.Now()
	var value string
	var err error

	// TODO: This is not a linearizable read.
	value, err = s.store.Get(in.GetKey())

	if err != nil {
		if errors.Is(err, store.ErrNoSuchKey) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.metrics.GrpcGet(in.GetKey(), time.Since(start).Seconds())
	return &pb.GetResp{
		Entry: &pb.Entry{
			Key:   in.GetKey(),
			Value: value,
		},
	}, nil
}

func (s *Server) Put(ctx context.Context, in *pb.PutReq) (*pb.PutResp, error) {
	start := time.Now()
	if len(in.GetKey()) > s.stCfg.MaxKeySize {
		return nil, status.Error(codes.InvalidArgument, store.ErrKeyTooLarge.Error())
	}
	if len(in.GetValue()) > s.stCfg.MaxValSize {
		return nil, status.Error(codes.InvalidArgument, store.ErrValueTooLarge.Error())
	}

	cmd := &fsm_v1.Command{
		Command: &fsm_v1.Command_Put{
			Put: &fsm_v1.PutCommand{
				Key:   in.GetKey(),
				Value: in.GetValue(),
			},
		},
	}
	data, err := proto.Marshal(cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal command")
	}

	res := s.raft.Submit(data)
	if !res.IsLeader {
		return nil, s.redirectErr(res)
	}

	promise := s.futures.NewPromise(res.LogIndex)
	if err := promise.Wait(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.metrics.GrpcPut(in.GetKey(), time.Since(start).Seconds())
	return &pb.PutResp{}, nil
}

func (s *Server) Delete(ctx context.Context, in *pb.DeleteReq) (*pb.DeleteResp, error) {
	start := time.Now()

	cmd := &fsm_v1.Command{
		Command: &fsm_v1.Command_Delete{
			Delete: &fsm_v1.DeleteCommand{
				Key: in.GetKey(),
			},
		},
	}
	data, err := proto.Marshal(cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal command")
	}

	res := s.raft.Submit(data)
	if !res.IsLeader {
		return nil, s.redirectErr(res)
	}

	promise := s.futures.NewPromise(res.LogIndex)
	if err := promise.Wait(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.metrics.GrpcDelete(in.GetKey(), time.Since(start).Seconds())
	return &pb.DeleteResp{}, nil
}

func (s *Server) redirectErr(res *raftapi.SubmitResult) error {
	if res.LeaderID >= 0 && res.LeaderID < len(s.raftPublicHTTPAddrs) {
		leaderAddr := s.raftPublicHTTPAddrs[res.LeaderID]
		return status.Errorf(codes.Unavailable, "not a leader, leader is at %s", leaderAddr)
	}
	return status.Error(codes.Unavailable, "no leader available")
}
