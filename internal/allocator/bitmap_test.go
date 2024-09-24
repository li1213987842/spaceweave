package allocator

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewBitMap(t *testing.T) {
	tests := []struct {
		name        string
		size        uint64
		shards      uint64
		expectedLen int
	}{
		{"Small BitMap", 640, 1, 1},
		{"Medium BitMap", 6400, 10, 10},
		{"Large BitMap", 64000, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm := NewBitMap(tt.size, tt.shards)
			if len(bm.shards) != tt.expectedLen {
				t.Errorf("NewBitMap() shard count = %v, want %v", len(bm.shards), tt.expectedLen)
			}
			for _, shard := range bm.shards {
				if len(shard.bits) != int(tt.size/64/tt.shards) {
					t.Errorf("NewBitMap() shard size = %v, want %v", len(shard.bits), tt.size/64/tt.shards)
				}
			}
		})
	}
}

func TestAllocateAndFree(t *testing.T) {
	bm := NewBitMap(640, 1) // 10 uint64 blocks

	// Allocate all space
	allocated := make([]uint64, 0, 10)
	for i := 0; i < 10; i++ {
		start, err := bm.Allocate(64)
		if err != nil {
			t.Errorf("Allocate() error = %v", err)
		}
		allocated = append(allocated, start)
	}

	// Try to allocate when full
	_, err := bm.Allocate(1)
	if err == nil {
		t.Errorf("Allocate() when full should return error")
	}

	// Free all allocated space
	for _, start := range allocated {
		err := bm.Free(start, 64)
		if err != nil {
			t.Errorf("Free() error = %v", err)
		}
	}

	// Check if all space is available
	if bm.GetAvailableSpace() != 640 {
		t.Errorf("After freeing all, available space = %v, want 640", bm.GetAvailableSpace())
	}
}

func TestConcurrentAllocateAndFree(t *testing.T) {
	bm := NewBitMap(64000, 100)
	concurrency := 100
	allocationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			allocated := make([]struct{ start, size uint64 }, 0, allocationsPerGoroutine)

			for j := 0; j < allocationsPerGoroutine; j++ {
				size := uint64(rand.Intn(63) + 1) // 1 to 64 bits
				start, err := bm.Allocate(size)
				if err == nil {
					allocated = append(allocated, struct{ start, size uint64 }{start, size})
				}

				if len(allocated) > 0 && rand.Float32() < 0.5 {
					// 50% chance to free a random allocation
					index := rand.Intn(len(allocated))
					toFree := allocated[index]
					err := bm.Free(toFree.start, toFree.size)
					if err != nil {
						t.Errorf("Free() error = %v", err)
					}
					// Remove the freed allocation from the slice
					allocated[index] = allocated[len(allocated)-1]
					allocated = allocated[:len(allocated)-1]
				}
			}

			// Free remaining allocations
			for _, a := range allocated {
				err := bm.Free(a.start, a.size)
				if err != nil {
					t.Errorf("Final Free() error = %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Check if all space is available after concurrent operations
	if bm.GetAvailableSpace() != 64000 {
		t.Errorf("After concurrent operations, available space = %v, want 64000", bm.GetAvailableSpace())
	}
}

func TestAllocationPerformance(t *testing.T) {
	bm := NewBitMap(1<<20, 100) // 1 million bits
	allocationSizes := []uint64{1, 64, 1024, 4096}

	for _, size := range allocationSizes {
		t.Run("AllocateSize_"+string(strconv.FormatInt(int64(size), 10)), func(t *testing.T) {
			start := time.Now()
			count := 0
			for time.Since(start) < time.Second {
				_, err := bm.Allocate(size)
				if err != nil {
					break
				}
				count++
			}
			t.Logf("Allocated %d blocks of size %d in 1 second", count, size)
		})

		// Reset bitmap for next test
		bm = NewBitMap(1<<20, 100)
	}
}
