package service

import (
	"context"
	"errors"

	"google.golang.org/grpc"

	pb "github.com/li1213987842/spaceweave/proto"
)

type Service struct {
	grpc *_GRPCService
}

func (s *Service) Initialize(ctx context.Context, gs *grpc.Server) error {
	if s.grpc != nil {
		return errors.New("service initialized")
	}

	grpc := &_GRPCService{s}
	pb.RegisterDiskAllocatorServer(gs, grpc)

	s.grpc = grpc
	return nil
}

func (s *Service) Finalize() error {
	return nil
}
