package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/li1213987842/spaceweave/config"
	"github.com/li1213987842/spaceweave/internal/allocator"
	"github.com/li1213987842/spaceweave/service"
)

func main() {
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		panic(fmt.Sprintf("load config from env fail: %v", err))
	}
	cinfo, err := json.Marshal(cfg)

	log.Println("load config info:", string(cinfo), err)
	service.ServConfig = cfg
	service.AllocatorStore = allocator.NewDiskAllocator(service.ServConfig)

	runService(service.ServConfig)
}

func runService(cfg *config.Config) {
	var opts []grpc.ServerOption
	opts = append(opts,
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Duration(cfg.GrpcMaxIdleSec) * time.Second,
			MaxConnectionAge:      0,
			MaxConnectionAgeGrace: 0,
			Time:                  7200 * time.Second,
			Timeout:               20 * time.Second,
		}),
	)
	//TODO tls...

	grpcServer := grpc.NewServer(opts...)

	spaceWeaveSvc := &service.Service{}
	if err := spaceWeaveSvc.Initialize(context.Background(), grpcServer); err != nil {
		log.Fatal("Failed to initialize space weave service", "err", err)
	}
	log.Println("space weave service initialized")
	defer spaceWeaveSvc.Finalize()

	grpcListener, err := net.Listen("tcp", cfg.SpaceWeaveAddr)
	if err != nil {
		log.Fatal("Failed to listen", "add", cfg.SpaceWeaveAddr, "err", err)
	}
	log.Println("gRPC service start listening", "addr", cfg.SpaceWeaveAddr)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			log.Println("Stopping gRPC service")
			grpcServer.GracefulStop()
			log.Println("gRPC service stopped")
			wg.Done()
		}()
	}()

	if err := grpcServer.Serve(grpcListener); err != nil {
		fmt.Printf("failed to serve: %v\n", err)
	}

}
