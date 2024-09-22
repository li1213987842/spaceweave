package allocator

import (
	"github.com/li1213987842/spaceweave/config"
	"github.com/li1213987842/spaceweave/internal/bitmap"
	"github.com/li1213987842/spaceweave/internal/rbtree"
)

const MiBThreshold = 64 //64 * 4KB = 256kb

type DiskAllocator interface {
	Allocate(size uint64) (uint64, error)
	Free(address uint64, size uint64) error
	GetDiskUtilization() float64
}

type diskAllocatorImpl struct {
	bitmaps *bitmap.ConcurrentBitMap
	tree    *rbtree.RBTree
	cfg     *config.Config
}

type UsageStats struct {
	UsedSpace  int64
	TotalSpace int64
	UsageRatio float64
}

func NewDiskAllocator(cfg *config.Config) DiskAllocator {
	return &diskAllocatorImpl{
		cfg:     cfg,
		bitmaps: bitmap.NewBitMap(cfg.SmallBlockLimit, cfg.NumShards),
		tree:    rbtree.NewRBTree(cfg.SmallBlockLimit, cfg.TotalSize/cfg.UnitSize-cfg.SmallBlockLimit),
	}
}

func (da *diskAllocatorImpl) Allocate(size uint64) (start uint64, err error) {
	units := (size + da.cfg.UnitSize - 1) / da.cfg.UnitSize // Round up to nearest unit
	if units <= MiBThreshold {
		start, err = da.allocateSmall(units)
		if err == nil {
			return start, nil
		}
	}
	start, err = da.allocateLarge(units)
	if err == nil {
		return start, nil
	}

	da.tree.Defragment()
	start, err = da.allocateLarge(units)
	if err == nil {
		return start, nil
	}

	return da.allocateSmall(units)
}

func (da *diskAllocatorImpl) allocateSmall(units uint64) (uint64, error) {
	start, err := da.bitmaps.Allocate(units)
	if err != nil {
		return 0, err
	}
	return start * da.cfg.UnitSize, nil
}

func (da *diskAllocatorImpl) allocateLarge(units uint64) (uint64, error) {
	start, err := da.tree.Allocate(units)
	if err != nil {
		return 0, err
	}
	return start * da.cfg.UnitSize, nil
}

func (da *diskAllocatorImpl) Free(address uint64, size uint64) error {
	start := address / da.cfg.UnitSize
	units := (size + da.cfg.UnitSize - 1) / da.cfg.UnitSize // Round up to nearest unit

	if start < da.cfg.SmallBlockLimit {
		blocks := units
		if start+blocks > da.cfg.SmallBlockLimit {
			blocks = da.cfg.SmallBlockLimit - start
		}
		da.bitmaps.Free(start, blocks)
		units -= blocks
		start += blocks
	}
	if start >= da.cfg.SmallBlockLimit && units > 0 {
		da.tree.Free(start, units)
	}
	return nil
}

func (da *diskAllocatorImpl) GetDiskUtilization() float64 {
	totalSpace := da.cfg.TotalSize
	availableSpace := (da.bitmaps.GetAvailableSpace() + da.tree.GetAvailableSpace()) * da.cfg.UnitSize
	usedSpace := totalSpace - availableSpace
	return float64(usedSpace) / float64(totalSpace)
}
