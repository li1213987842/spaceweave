package allocator

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/btree"

	"github.com/li1213987842/spaceweave/config"
)

type persistentData struct {
	Bitmaps  [][]uint64
	TreeData []BTreeBlock
}

func (da *diskAllocatorImpl) SaveState() error {
	data := persistentData{
		Bitmaps:  make([][]uint64, len(da.bitmaps.shards)),
		TreeData: make([]BTreeBlock, 0),
	}

	// Save bitmap data
	for i, shard := range da.bitmaps.shards {
		shard.mu.RLock()
		data.Bitmaps[i] = make([]uint64, len(shard.bits))
		copy(data.Bitmaps[i], shard.bits)
		shard.mu.RUnlock()
	}

	aval := uint64(0)
	// Save btree data
	da.tree.mu.RLock()
	da.tree.treeByStart.Ascend(func(item btree.Item) bool {
		block := item.(BlockByStart).BTreeBlock
		data.TreeData = append(data.TreeData, *block)
		aval += block.Size
		return true
	})
	da.tree.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(da.cfg.StatePersistencePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Open file for writing
	file, err := os.Create(da.cfg.StatePersistencePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode and write data
	encoder := gob.NewEncoder(file)
	return encoder.Encode(data)
}

func LoadState(cfg *config.Config) (DiskAllocator, error) {
	da := &diskAllocatorImpl{
		cfg:            cfg,
		bitmaps:        NewBitMap(cfg.SmallBlockLimit, cfg.NumShards),
		tree:           NewBTreeManager(cfg.TotalSize/cfg.UnitSize - cfg.SmallBlockLimit),
		lastBackupTime: time.Now(),
		closeChan:      make(chan struct{}),
	}

	// No state persistence
	if cfg.StatePersistencePath == "" {
		return da, nil
	}

	if _, err := os.Stat(cfg.StatePersistencePath); os.IsNotExist(err) {
		da.startBackupRoutine()
		return da, nil
	}

	// Open file for reading
	file, err := os.Open(cfg.StatePersistencePath)
	if err != nil {
		fmt.Println("dddd", err)
		return nil, err
	}
	defer file.Close()
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	// Check if file is empty
	if fileInfo.Size() == 0 {
		return da, nil
	}
	// Decode data
	var data persistentData
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// Restore bitmap data
	if len(data.Bitmaps) != len(da.bitmaps.shards) {
		return nil, fmt.Errorf("mismatch in number of bitmap shards")
	}
	for i, bits := range data.Bitmaps {
		if len(bits) != len(da.bitmaps.shards[i].bits) {
			return nil, fmt.Errorf("mismatch in bitmap size for shard %d", i)
		}
		copy(da.bitmaps.shards[i].bits, bits)
	}
	// Restore btree data
	da.tree = NewBTreeManagerWithBlocks(cfg.TotalSize/cfg.UnitSize-cfg.SmallBlockLimit, data.TreeData)
	da.startBackupRoutine()
	return da, nil
}
