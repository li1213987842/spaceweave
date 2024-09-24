package allocator

import (
	"testing"
)

func TestNewDiskManager(t *testing.T) {
	totalSpace := uint64(1024)
	dm := NewBTreeManager(totalSpace)

	if dm.totalSpace != totalSpace {
		t.Errorf("Expected total space %d, got %d", totalSpace, dm.totalSpace)
	}

	if dm.freeSpace != totalSpace {
		t.Errorf("Expected free space %d, got %d", totalSpace, dm.freeSpace)
	}

	if dm.GetAvailableSpace() != totalSpace {
		t.Errorf("Expected available space %d, got %d", totalSpace, dm.GetAvailableSpace())
	}
}

func TestAllocate(t *testing.T) {
	dm := NewBTreeManager(1024)

	tests := []struct {
		name          string
		size          uint64
		expectedStart uint64
		expectedError bool
	}{
		{"Allocate half", 512, 0, false},
		{"Allocate quarter", 256, 512, false},
		{"Allocate remaining", 256, 768, false},
		{"Allocate too much", 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, err := dm.Allocate(tt.size)
			if (err != nil) != tt.expectedError {
				t.Errorf("Allocate() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !tt.expectedError && start != tt.expectedStart {
				t.Errorf("Allocate() start = %v, want %v", start, tt.expectedStart)
			}
		})
	}

	if dm.GetAvailableSpace() != 0 {
		t.Errorf("Expected available space 0, got %d", dm.GetAvailableSpace())
	}
}

func TestFree(t *testing.T) {
	dm := NewBTreeManager(1024)

	// Allocate some space
	start1, _ := dm.Allocate(256)
	start2, _ := dm.Allocate(256)
	start3, _ := dm.Allocate(256)

	// Free the middle block
	err := dm.Free(start2, 256)
	if err != nil {
		t.Errorf("Free() error = %v", err)
	}

	if dm.GetAvailableSpace() != 512 {
		t.Errorf("Expected available space 512, got %d", dm.GetAvailableSpace())
	}

	// Free the first block (should merge with the second)
	err = dm.Free(start1, 256)
	if err != nil {
		t.Errorf("Free() error = %v", err)
	}

	if dm.GetAvailableSpace() != 768 {
		t.Errorf("Expected available space 768, got %d", dm.GetAvailableSpace())
	}

	// Free the last block (should merge everything)
	err = dm.Free(start3, 256)
	if err != nil {
		t.Errorf("Free() error = %v", err)
	}

	if dm.GetAvailableSpace() != 1024 {
		t.Errorf("Expected available space 1024, got %d", dm.GetAvailableSpace())
	}
}

func TestAllocateAfterFree(t *testing.T) {
	dm := NewBTreeManager(1024)

	// Allocate all space
	dm.Allocate(1024)

	// Free some space
	dm.Free(512, 256)

	// Try to allocate more than available
	_, err := dm.Allocate(512)
	if err == nil {
		t.Errorf("Expected error when allocating more than available")
	}

	// Allocate available space
	start, err := dm.Allocate(256)
	if err != nil {
		t.Errorf("Allocate() error = %v", err)
	}
	if start != 512 {
		t.Errorf("Expected start 512, got %d", start)
	}

	if dm.GetAvailableSpace() != 0 {
		t.Errorf("Expected available space 0, got %d", dm.GetAvailableSpace())
	}
}

func TestBTreeConcurrentAllocateAndFree(t *testing.T) {
	dm := NewBTreeManager(1024 * 1024) // 1 MiB
	concurrency := 100
	allocSize := uint64(1024) // 1 KiB

	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				start, err := dm.Allocate(allocSize)
				if err != nil {
					t.Errorf("Allocate() error = %v", err)
				}
				dm.Free(start, allocSize)
			}
			done <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}

	if dm.GetAvailableSpace() != 1024*1024 {
		t.Errorf("Expected available space 1024*1024, got %d", dm.GetAvailableSpace())
	}
}
