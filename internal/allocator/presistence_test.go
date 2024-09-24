package allocator

import (
	"math"
	"os"
	"testing"

	"github.com/li1213987842/spaceweave/config"
)

func TestSaveAndLoadStateDetailed(t *testing.T) {
	// Create a temporary file for state persistence
	tmpfile, err := os.CreateTemp("", "test-state-*.gob")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Create a test configuration
	cfg := &config.Config{
		UnitSize:             4096,
		TotalSize:            1024 * 1024 * 1024, // 1 GB
		SmallBlockLimit:      1024,
		NumShards:            16,
		StatePersistencePath: tmpfile.Name(),
		BackupIntervalSec:    5,
	}

	// Create a new disk allocator
	da := NewDiskAllocator(cfg)

	// Perform some allocations
	allocations := []struct {
		size uint64
		addr uint64
	}{
		{size: 1024 * 1024, addr: 0},     // 1 MB
		{size: 512 * 1024, addr: 0},      // 512 KB
		{size: 2 * 1024 * 1024, addr: 0}, // 2 MB
	}

	for i := range allocations {
		addr, err := da.Allocate(allocations[i].size)
		if err != nil {
			t.Fatalf("Failed to allocate %d bytes: %v", allocations[i].size, err)
		}
		allocations[i].addr = addr
	}

	// Free one allocation
	err = da.Free(allocations[1].addr, allocations[1].size)
	if err != nil {
		t.Fatalf("Failed to free allocation: %v", err)
	}

	// Save the state
	err = da.SaveState()
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Close the current allocator
	err = da.Close()
	if err != nil {
		t.Fatalf("Failed to close allocator: %v", err)
	}

	// Load the state into a new allocator
	loadedDA, err := LoadState(cfg)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify the loaded state
	t.Run("VerifyLoadedState", func(t *testing.T) {
		// Check disk utilization
		expectedUtilization := float64(allocations[0].size+allocations[2].size) / float64(cfg.TotalSize)
		actualUtilization := loadedDA.GetDiskUtilization()
		if math.Abs(expectedUtilization-actualUtilization) > 0.001 {
			t.Errorf("Disk utilization mismatch: expected %f, got %f", expectedUtilization, actualUtilization)
		}

		// Try to allocate the previously freed space
		addr, err := loadedDA.Allocate(allocations[1].size)
		if err != nil {
			t.Errorf("Failed to allocate previously freed space: %v", err)
		}
		if addr != allocations[1].addr {
			t.Errorf("Expected to allocate at %d, but got %d", allocations[1].addr, addr)
		}

		// Verify we can't allocate more than available
		_, err = loadedDA.Allocate(cfg.TotalSize)
		if err == nil {
			t.Error("Should not be able to allocate more than total size")
		}
	})

	// Test freeing and reallocating across restarts
	t.Run("FreeAndReallocateAcrossRestarts", func(t *testing.T) {
		// Free a large allocation
		err = loadedDA.Free(allocations[2].addr, allocations[2].size)
		if err != nil {
			t.Fatalf("Failed to free large allocation: %v", err)
		}

		// Save and reload state
		err = loadedDA.SaveState()
		if err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}
		err = loadedDA.Close()
		if err != nil {
			t.Fatalf("Failed to close allocator: %v", err)
		}

		reloadedDA, err := LoadState(cfg)
		if err != nil {
			t.Fatalf("Failed to reload state: %v", err)
		}

		// Try to reallocate the freed space
		newAddr, err := reloadedDA.Allocate(allocations[2].size)
		if err != nil {
			t.Errorf("Failed to reallocate freed space: %v", err)
		}
		if newAddr != allocations[2].addr {
			t.Errorf("Expected to reallocate at %d, but got %d", allocations[2].addr, newAddr)
		}

		err = reloadedDA.Close()
		if err != nil {
			t.Fatalf("Failed to close reloaded allocator: %v", err)
		}
	})
}
