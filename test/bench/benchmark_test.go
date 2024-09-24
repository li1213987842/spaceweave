package bench

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/li1213987842/spaceweave/config"
	"github.com/li1213987842/spaceweave/internal/allocator"
)

const (
	TiB            = 1024 * 1024 * 1024 * 1024
	MaxRequestSize = 4 * 1024 * 1024 // 4 MiB
	MinRequestSize = 512             // 512 Bytes
	NumGoroutines  = 16              // 并发goroutine数量
	BatchSize      = 1000            // 批处理大小
)

type Operation struct {
	size    uint64
	address uint64
}

// BenchmarkStats 用于存储基准测试的统计信息
type BenchmarkStats struct {
	AllocOps, FreeOps        int64
	AllocTime, FreeTime      int64
	TotalWritten, TotalFreed uint64
}

// 使用快速随机数生成器
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func runConcurrentBenchmark(b *testing.B, targetWrite uint64) {
	cfg := &config.Config{
		TotalSize:            targetWrite,
		UnitSize:             4 * 1024,
		NumShards:            256,
		SmallBlockLimit:      uint64(float64(targetWrite)*0.1) / (4 * 1024),
		StatePersistencePath: "",
	}
	rand.Seed(time.Now().UnixNano())

	allocator := allocator.NewDiskAllocator(cfg)
	defer allocator.Close()
	stats := &BenchmarkStats{}

	freeRatio := rng.Float64()*0.3 + 0.2 // 20% to 50%
	start := time.Now()

	operations := writeFull(b, allocator, stats, freeRatio)
	freeOperations(b, allocator, operations, stats)
	writeFull(b, allocator, stats, freeRatio)

	totalTime := time.Since(start)
	utilization := allocator.GetDiskUtilization()

	printBenchmarkResults(b, totalTime, stats, utilization)
}

func writeFull(b *testing.B, allocator allocator.DiskAllocator, stats *BenchmarkStats, freeRatio float64) []Operation {
	var wg sync.WaitGroup
	operationsChan := make(chan Operation, NumGoroutines*BatchSize)
	done := make(chan struct{})

	for i := 0; i < NumGoroutines; i++ {
		wg.Add(1)
		go writeWorker(allocator, operationsChan, done, stats, freeRatio, &wg)
	}

	go func() {
		wg.Wait()
		close(operationsChan)
		close(done)
	}()

	return collectOperations(operationsChan)
}

func writeWorker(allocator allocator.DiskAllocator, operationsChan chan<- Operation, done <-chan struct{}, stats *BenchmarkStats, freeRatio float64, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-done:
			return
		default:
			localStats := struct {
				allocOps, allocTime int64
				written             uint64
			}{}

			for j := 0; j < BatchSize; j++ {
				size := MinRequestSize + rand.Uint64()%(MaxRequestSize-MinRequestSize+1)
				size = ((size + 511) / 512) * 512 // 调整为 512 字节的倍数

				start := time.Now()
				address, err := allocator.Allocate(size)
				duration := time.Since(start)
				if err != nil {
					return
				}
				localStats.allocOps++
				localStats.allocTime += duration.Milliseconds()
				localStats.written += size
				if rng.Float64() < freeRatio {
					operationsChan <- Operation{size: size, address: address}
				}
			}

			atomic.AddInt64(&stats.AllocOps, localStats.allocOps)
			atomic.AddInt64(&stats.AllocTime, localStats.allocTime)
			atomic.AddUint64(&stats.TotalWritten, localStats.written)
		}
	}
}

func collectOperations(operationsChan <-chan Operation) []Operation {
	operations := make([]Operation, 0, NumGoroutines*BatchSize)
	for op := range operationsChan {
		operations = append(operations, op)
	}
	return operations
}

func freeOperations(b *testing.B, allocator allocator.DiskAllocator, operations []Operation, stats *BenchmarkStats) {
	toFree := len(operations)
	var wg sync.WaitGroup
	opsChan := make(chan Operation, toFree)

	for _, op := range operations {
		opsChan <- op
	}
	close(opsChan)

	for i := 0; i < NumGoroutines; i++ {
		wg.Add(1)
		go freeWorker(b, allocator, opsChan, stats, toFree/NumGoroutines, &wg)
	}

	wg.Wait()
}

func freeWorker(b *testing.B, allocator allocator.DiskAllocator, opsChan <-chan Operation, stats *BenchmarkStats, limit int, wg *sync.WaitGroup) {
	defer wg.Done()
	localStats := struct {
		freeOps, freeTime int64
		freed             uint64
	}{}

	for op := range opsChan {
		if localStats.freeOps >= int64(limit) {
			break
		}
		start := time.Now()
		err := allocator.Free(op.address, op.size)
		duration := time.Since(start)
		if err != nil {
			b.Logf("Free failed: %v", err)
			continue
		}
		localStats.freeOps++
		localStats.freeTime += duration.Milliseconds()
		localStats.freed += op.size
	}

	atomic.AddInt64(&stats.FreeOps, localStats.freeOps)
	atomic.AddInt64(&stats.FreeTime, localStats.freeTime)
	atomic.AddUint64(&stats.TotalFreed, localStats.freed)
}

func randomSize() uint64 {
	sizes := []uint64{
		4 * 1024,        // 4KB
		16 * 1024,       // 16KB
		64 * 1024,       // 64KB
		256 * 1024,      // 256KB
		1 * 1024 * 1024, // 1MB
		4 * 1024 * 1024, // 4MB
	}
	return sizes[rand.Intn(len(sizes))]
}
func printBenchmarkResults(b *testing.B, totalTime time.Duration, stats *BenchmarkStats, utilization float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	b.ReportMetric(float64(stats.TotalWritten)/float64(TiB), "TiB_written")
	b.ReportMetric(float64(stats.TotalFreed)/float64(TiB), "TiB_free")
	b.ReportMetric(utilization*100, "disk_utilization_%")
	b.ReportMetric(float64(stats.AllocTime)/float64(stats.AllocOps), "avg_alloc_time_us")
	b.ReportMetric(float64(stats.FreeTime)/float64(stats.FreeOps), "avg_free_time_us")
	b.ReportMetric(totalTime.Seconds(), "total_time_sec")
	b.ReportMetric(float64(m.Alloc)/float64(1024*1024), "memory_usage_MiB")
}

func BenchmarkConcurrentDiskAllocator(b *testing.B) {
	sizes := []struct {
		name string
		size uint64
	}{
		{"10TiB", 10 * TiB},
		{"50TiB", 50 * TiB},
		{"100TiB", 100 * TiB},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			runConcurrentBenchmark(b, s.size)
		})
	}
}
