package service

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/li1213987842/spaceweave/proto"
)

type _GRPCService struct {
	s *Service
}

func (s *_GRPCService) Allocate(ctx context.Context, req *pb.AllocateRequest) (resp *pb.AllocateResponse, err error) {
	if req.Size <= 0 {
		return nil, errors.New(fmt.Sprintf("Invalid Argument: size %d", req.Size))
	}
	addr, err := AllocatorStore.Allocate(req.Size)
	if err != nil {
		return nil, err
	}
	return &pb.AllocateResponse{Address: addr}, nil
}

func (s *_GRPCService) Free(ctx context.Context, req *pb.FreeRequest) (resp *pb.FreeResponse, err error) {
	return &pb.FreeResponse{}, AllocatorStore.Free(req.Address, req.Size)
}

func (s *_GRPCService) GetDiskUtilization(ctx context.Context, req *pb.GetDiskUtilizationRequest) (resp *pb.GetDiskUtilizationResponse, err error) {
	utilization := AllocatorStore.GetDiskUtilization()
	return &pb.GetDiskUtilizationResponse{Utilization: float32(utilization)}, nil
}
