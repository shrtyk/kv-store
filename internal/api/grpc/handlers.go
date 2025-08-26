package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/shrtyk/kv-store/internal/core/ports/store"
	pb "github.com/shrtyk/kv-store/proto/grpc/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) Get(ctx context.Context, in *pb.GetReq) (*pb.GetResp, error) {
	start := time.Now()
	value, err := s.store.Get(in.GetKey())
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

	s.tl.WritePut(in.GetKey(), in.GetValue())
	err := s.store.Put(in.GetKey(), in.GetValue())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.metrics.GrpcPut(in.GetKey(), time.Since(start).Seconds())
	return &pb.PutResp{}, nil
}

func (s *Server) Delete(ctx context.Context, in *pb.DeleteReq) (*pb.DeleteResp, error) {
	start := time.Now()
	s.tl.WriteDelete(in.GetKey())
	err := s.store.Delete(in.GetKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.metrics.GrpcDelete(in.GetKey(), time.Since(start).Seconds())
	return &pb.DeleteResp{}, nil
}
