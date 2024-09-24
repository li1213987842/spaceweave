package allocator

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/li1213987842/spaceweave/config"
)

const MiBThreshold = 64 //64 * 4KB = 256kb

type DiskAllocator interface {
	Allocate(size uint64) (uint64, error)
	Free(address uint64, size uint64) error
	GetDiskUtilization() float64
	SaveState() error
	Close() error
}

type diskAllocatorImpl struct {
	bitmaps *ConcurrentBitMap
	tree    *BTreeManager
	cfg     *config.Config

	operationCount        int64
	lastBackupTime        time.Time
	lastBackupUtilization float64

	closeChan chan struct{}
	closeWg   sync.WaitGroup
}

type UsageStats struct {
	UsedSpace  int64
	TotalSpace int64
	UsageRatio float64
}

func NewDiskAllocator(cfg *config.Config) DiskAllocator {
	da, err := LoadState(cfg)
	if err != nil {
		panic(err)
	}
	return da
}

func (da *diskAllocatorImpl) startBackupRoutine() {
	da.closeWg.Add(1)
	go func() {
		defer da.closeWg.Done()
		ticker := time.NewTicker(time.Duration(da.cfg.BackupIntervalSec) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				da.checkAndBackup()
			case <-da.closeChan:
				return
			}
		}
	}()
}

func (da *diskAllocatorImpl) checkAndBackup() {
	operationCount := atomic.LoadInt64(&da.operationCount)

	if uint64(operationCount) >= da.cfg.BackupOperationThreshold ||
		time.Since(da.lastBackupTime) >= time.Duration(da.cfg.BackupIntervalSec)*time.Second {
		atomic.AddInt64(&da.operationCount, -operationCount) // Reset operation count

		err := da.SaveState()
		if err == nil {
			da.operationCount = 0
			da.lastBackupTime = time.Now()
		}
	}
}

func (da *diskAllocatorImpl) incrementOperationCount() {
	atomic.AddInt64(&da.operationCount, 1)
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

	return da.allocateSmall(units)
}

func (da *diskAllocatorImpl) allocateSmall(units uint64) (uint64, error) {
	start, err := da.bitmaps.Allocate(units)
	if err != nil {
		return 0, err
	}
	da.incrementOperationCount()
	return start * da.cfg.UnitSize, nil
}

func (da *diskAllocatorImpl) allocateLarge(units uint64) (uint64, error) {
	start, err := da.tree.Allocate(units)
	if err != nil {
		return 0, err
	}
	da.incrementOperationCount()
	return (start + da.cfg.SmallBlockLimit) * da.cfg.UnitSize, nil
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
		da.tree.Free(start-da.cfg.SmallBlockLimit, units)
	}
	da.incrementOperationCount()
	return nil
}

func (da *diskAllocatorImpl) GetDiskUtilization() float64 {
	totalSpace := da.cfg.TotalSize
	availableSpace := (da.bitmaps.GetAvailableSpace() + da.tree.GetAvailableSpace()) * da.cfg.UnitSize
	usedSpace := totalSpace - availableSpace
	return float64(usedSpace) / float64(totalSpace)
}

func (da *diskAllocatorImpl) Close() error {
	close(da.closeChan)
	da.closeWg.Wait()
	return da.SaveState() // Final backup on close
}
