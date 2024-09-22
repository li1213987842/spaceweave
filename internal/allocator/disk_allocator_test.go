package allocator

import (
	"testing"

	"github.com/li1213987842/spaceweave/config"
)

func TestDiskAllocator(t *testing.T) {
	cfg := &config.Config{
		TotalSize:       100 * 1024 * 1024 * 1024, // 1GB
		UnitSize:        4096,                     // 4KB
		SmallBlockLimit: 1024 * 256,               // 1MB
		NumShards:       16,
	}

	da := NewDiskAllocator(cfg)

	t.Run("Allocate and Free Small Block", func(t *testing.T) {
		size := uint64(4096) // 4KB
		addr, err := da.Allocate(size)
		if err != nil {
			t.Fatalf("Failed to allocate small block: %v", err)
		}
		if addr%cfg.UnitSize != 0 {
			t.Errorf("Allocated address not aligned: %d", addr)
		}
		err = da.Free(addr, size)
		if err != nil {
			t.Fatalf("Failed to free small block: %v", err)
		}
	})

	t.Run("Allocate and Free Large Block", func(t *testing.T) {
		size := uint64(2 * 1024 * 1024) // 2MB
		addr, err := da.Allocate(size)
		if err != nil {
			t.Fatalf("Failed to allocate large block: %v", err)
		}
		if addr%cfg.UnitSize != 0 {
			t.Errorf("Allocated address not aligned: %d", addr)
		}
		err = da.Free(addr, size)
		if err != nil {
			t.Fatalf("Failed to free large block: %v", err)
		}
	})

	t.Run("Allocate Until Full", func(t *testing.T) {
		allocatedAddresses := make([]uint64, 0)
		allocatedSizes := make([]uint64, 0)

		for {
			size := uint64(1024 * 1024) // 1MB
			addr, err := da.Allocate(size)
			if err != nil {
				break // Expected to fail when full
			}
			allocatedAddresses = append(allocatedAddresses, addr)
			allocatedSizes = append(allocatedSizes, size)
		}

		utilization := da.GetDiskUtilization()
		if utilization < 0.99 {
			t.Errorf("Expected utilization to be close to 1, got %f", utilization)
		}

		// Free all allocated blocks
		for i, addr := range allocatedAddresses {
			err := da.Free(addr, allocatedSizes[i])
			if err != nil {
				t.Fatalf("Failed to free block: %v", err)
			}
		}

		utilization = da.GetDiskUtilization()
		if utilization > 0.01 {
			t.Errorf("Expected utilization to be close to 0, got %f", utilization)
		}
	})

	t.Run("Allocation Alignment", func(t *testing.T) {
		sizes := []uint64{1, 4095, 4096, 4097, 8192}
		for _, size := range sizes {
			addr, err := da.Allocate(size)
			if err != nil {
				t.Fatalf("Failed to allocate %d bytes: %v", size, err)
			}
			if addr%cfg.UnitSize != 0 {
				t.Errorf("Allocated address not aligned for size %d: %d", size, addr)
			}
			da.Free(addr, size)
		}
	})
}
