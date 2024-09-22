package bitmap

import (
	"sync"
	"testing"
)

const (
	TB            = 1024 * 1024 * 1024 * 1024
	testSpaceSize = TB
	testShards    = 1024
	blockSize     = 4 * 1024 // 4KB
	bitsPerBlock  = 1        // Each bit in the bitmap represents one 4KB block
	totalBlocks   = testSpaceSize / blockSize
)

func TestConcurrentBitMap(t *testing.T) {
	t.Run("Basic Allocation and Free", func(t *testing.T) {
		bm := NewBitMap(totalBlocks, testShards)

		start, err := bm.Allocate(1) // Allocate 1 block (4KB)
		if err != nil {
			t.Fatalf("Failed to allocate: %v", err)
		}
		if start != 0 {
			t.Errorf("Expected start to be 0, got %d", start)
		}

		err = bm.Free(start, 1)
		if err != nil {
			t.Fatalf("Failed to free: %v", err)
		}

		available := bm.GetAvailableSpace()
		if available != totalBlocks {
			t.Errorf("Expected available space to be %d blocks, got %d", totalBlocks, available)
		}
	})

	t.Run("Multiple Allocations", func(t *testing.T) {
		bm := NewBitMap(totalBlocks, testShards)

		allocations := []uint64{1, 2, 4, 8, 16} // In blocks (4KB, 8KB, 16KB, 32KB, 64KB)
		starts := make([]uint64, len(allocations))

		for i, size := range allocations {
			start, err := bm.Allocate(size)
			if err != nil {
				t.Fatalf("Failed to allocate %d blocks: %v", size, err)
			}
			starts[i] = start
		}

		for i, start := range starts {
			err := bm.Free(start, allocations[i])
			if err != nil {
				t.Fatalf("Failed to free allocation at %d with size %d blocks: %v", start, allocations[i], err)
			}
		}

		available := bm.GetAvailableSpace()
		if available != totalBlocks {
			t.Errorf("Expected available space to be %d blocks, got %d", totalBlocks, available)
		}
	})

	t.Run("Out of Space", func(t *testing.T) {
		bm := NewBitMap(64, 1) // Only 64 blocks available

		_, err := bm.Allocate(65) // Try to allocate more than available
		if err == nil {
			t.Error("Expected an error when allocating more than available space")
		}
	})

	t.Run("Concurrent Allocations", func(t *testing.T) {
		bm := NewBitMap(totalBlocks, testShards)
		var wg sync.WaitGroup
		allocations := 1000
		allocationSize := uint64(1) // 1 block (4KB)
		results := make(chan bool, allocations)

		for i := 0; i < allocations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start, err := bm.Allocate(allocationSize)
				if err != nil {
					results <- false
					return
				}
				results <- true
				bm.Free(start, allocationSize)
			}()
		}

		wg.Wait()
		close(results)

		successCount := 0
		for success := range results {
			if success {
				successCount++
			}
		}

		if successCount != allocations {
			t.Errorf("Expected %d successful allocations, got %d", allocations, successCount)
		}

		available := bm.GetAvailableSpace()
		if available != totalBlocks {
			t.Errorf("Expected available space to be %d blocks, got %d", totalBlocks, available)
		}
	})

	t.Run("Large Allocation and Free", func(t *testing.T) {
		bm := NewBitMap(totalBlocks, testShards)

		largeSize := uint64(64) // 64 blocks (256KB)
		start, err := bm.Allocate(largeSize)
		if err != nil {
			t.Fatalf("Failed to allocate large block: %v", err)
		}

		err = bm.Free(start, largeSize)
		if err != nil {
			t.Fatalf("Failed to free large block: %v", err)
		}

		available := bm.GetAvailableSpace()
		if available != totalBlocks {
			t.Errorf("Expected available space to be %d blocks, got %d", totalBlocks, available)
		}
	})
}
