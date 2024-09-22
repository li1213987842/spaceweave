package client

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "github.com/li1213987842/spaceweave/proto"
)

var (
	ErrInvalid = errors.New("invalid")
)

type DiskAllocatorClient interface {
	Allocate(ctx context.Context, size uint64) (uint64, error)
	Free(ctx context.Context, address uint64, size uint64) error
	GetDiskUtilization(ctx context.Context) (float32, error)
	Close() error
}

var _ DiskAllocatorClient = (*diskAllocatorClientImpl)(nil)

type diskAllocatorClientImpl struct {
	client pb.DiskAllocatorClient
	conn   *grpc.ClientConn
}

func NewDiskAllocatorClient(ctx context.Context, serverAddr string) (DiskAllocatorClient, error) {
	if serverAddr == "" {
		return nil, errors.Wrap(ErrInvalid, "server addr is empty")
	}

	var dialOpts []grpc.DialOption
	// --todo: add tls dial option and interceptors
	dialOpts = append(dialOpts, grpc.WithInsecure())

	conn, err := grpc.DialContext(ctx, serverAddr, dialOpts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "dial %s", serverAddr)
	}

	return &diskAllocatorClientImpl{
		client: pb.NewDiskAllocatorClient(conn),
		conn:   conn,
	}, nil
}

func (c *diskAllocatorClientImpl) Close() error {
	return c.conn.Close()
}

func (c *diskAllocatorClientImpl) Allocate(ctx context.Context, size uint64) (uint64, error) {
	r, err := c.client.Allocate(ctx, &pb.AllocateRequest{Size: size})
	if err != nil {
		return 0, err
	}
	return r.Address, nil
}

func (c *diskAllocatorClientImpl) Free(ctx context.Context, address uint64, size uint64) error {
	_, err := c.client.Free(ctx, &pb.FreeRequest{Address: address, Size: size})
	return err
}

func (c *diskAllocatorClientImpl) GetDiskUtilization(ctx context.Context) (float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := c.client.GetDiskUtilization(ctx, &pb.GetDiskUtilizationRequest{})
	if err != nil {
		return 0, err
	}
	return res.Utilization, nil
}
