package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/li1213987842/spaceweave/client"
)

const (
	serverAddr        = "localhost:22500" // 假设服务器地址
	minRequestSize    = 512
	maxRequestSize    = 4 * 1024 * 1024 // 4 MiB
	deleteProbability = 0.2             // 20% 的概率执行删除操作
	concurrentClients = 16
)

type allocation struct {
	address uint64
	size    uint64
}

func main() {
	rand.Seed(time.Now().UnixNano())
	c, err := client.NewDiskAllocatorClient(context.Background(), serverAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	var (
		allocations      []allocation
		totalWritten     uint64
		allocationsMutex sync.Mutex
		wg               sync.WaitGroup
	)

	startTime := time.Now()
	for i := 0; i < concurrentClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				ctx := context.Background()
				if rand.Float32() < deleteProbability && len(allocations) > 0 {
					// 删除操作
					allocationsMutex.Lock()
					index := rand.Intn(len(allocations))
					alloc := allocations[index]
					allocations = append(allocations[:index], allocations[index+1:]...)
					allocationsMutex.Unlock()

					err := c.Free(ctx, alloc.address, alloc.size)
					if err != nil {
						log.Printf("Failed to free space: %v", err)
					}
				} else {
					size := minRequestSize + rand.Uint64()%(maxRequestSize-minRequestSize+1)
					size = ((size + 511) / 512) * 512 // 调整为 512 字节的倍数

					address, err := c.Allocate(ctx, size)
					if err != nil {
						utilization, _ := c.GetDiskUtilization(ctx)
						log.Printf("Disk full. Utilization: %.2f%%", utilization*100)
						return
					}

					allocationsMutex.Lock()
					allocations = append(allocations, allocation{address, size})
					totalWritten += size
					allocationsMutex.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	duration := time.Since(startTime)
	utilization, _ := c.GetDiskUtilization(context.Background())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Test completed in %v\n", duration)
	fmt.Printf("Total data written: %.2f GiB\n", float64(totalWritten)/float64(1024*1024*1024))
	fmt.Printf("Disk utilization: %.2f%%\n", utilization*100)
	fmt.Printf("Estimated memory usage: %.2f MiB\n", float64(m.Alloc)/float64(1024*1024))
}
