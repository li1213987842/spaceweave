package rbtree

import (
	"testing"
)

func TestRBTree(t *testing.T) {
	t.Run("Basic Operations", func(t *testing.T) {
		tree := NewRBTree(0, 1000)

		// Test Allocate
		start, err := tree.Allocate(100)
		if err != nil || start != 0 {
			t.Errorf("Allocation failed or returned unexpected start: %v, %v", start, err)
		}

		// Test Free
		err = tree.Free(start, 100)
		if err != nil {
			t.Errorf("Free failed: %v", err)
		}

		// Test multiple allocations and frees
		allocations := []uint64{200, 300, 150}
		starts := make([]uint64, len(allocations))
		for i, size := range allocations {
			starts[i], err = tree.Allocate(size)
			if err != nil {
				t.Errorf("Allocation %d failed: %v", i, err)
			}
		}

		for i, start := range starts {
			err = tree.Free(start, allocations[i])
			if err != nil {
				t.Errorf("Free %d failed: %v", i, err)
			}
		}
	})

	t.Run("Best Fit Allocation", func(t *testing.T) {
		tree := NewRBTree(0, 1000)

		// Create some fragmentation
		tree.Allocate(200)
		mid, _ := tree.Allocate(200)
		tree.Allocate(200)
		tree.Free(mid, 200)

		// Test if it finds the best fit
		start, err := tree.Allocate(150)
		if err != nil || start != mid {
			t.Errorf("Best fit allocation failed or chose wrong block: %v, %v", start, err)
		}
	})

	t.Run("Out of Memory", func(t *testing.T) {
		tree := NewRBTree(0, 1000)

		// Allocate all memory
		_, err := tree.Allocate(1000)
		if err != nil {
			t.Errorf("Failed to allocate all memory: %v", err)
		}

		// Try to allocate more
		_, err = tree.Allocate(1)
		if err == nil {
			t.Error("Expected out of memory error, got nil")
		}
	})

	t.Run("Merging Free Blocks", func(t *testing.T) {
		tree := NewRBTree(0, 1000)

		// Allocate three blocks
		start1, _ := tree.Allocate(300)
		start2, _ := tree.Allocate(300)
		start3, _ := tree.Allocate(300)

		// Free them in reverse order
		tree.Free(start3, 300)
		tree.Free(start2, 300)
		tree.Free(start1, 300)

		// Try to allocate the entire space
		start, err := tree.Allocate(900)
		if err != nil || start != 0 {
			t.Errorf("Failed to allocate merged space: %v, %v", start, err)
		}
	})

	t.Run("GetAvailableSpace", func(t *testing.T) {
		tree := NewRBTree(0, 1000)

		if space := tree.GetAvailableSpace(); space != 1000 {
			t.Errorf("Initial available space incorrect: %v", space)
		}

		tree.Allocate(300)
		if space := tree.GetAvailableSpace(); space != 700 {
			t.Errorf("Available space after allocation incorrect: %v", space)
		}

		tree.Free(0, 300)
		if space := tree.GetAvailableSpace(); space != 1000 {
			t.Errorf("Available space after free incorrect: %v", space)
		}
	})
}
