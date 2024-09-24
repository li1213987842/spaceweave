package allocator

import (
	"errors"
	"sync"

	"github.com/google/btree"
)

type BTreeBlock struct {
	Start uint64
	Size  uint64
}

type BlockBySize struct {
	*BTreeBlock
}

func (b BlockBySize) Less(than btree.Item) bool {
	return b.Size < than.(BlockBySize).Size
}

type BlockByStart struct {
	*BTreeBlock
}

func (b BlockByStart) Less(than btree.Item) bool {
	return b.Start < than.(BlockByStart).Start
}

type BTreeManager struct {
	treeBySize  *btree.BTree
	treeByStart *btree.BTree
	mu          sync.RWMutex
	totalSpace  uint64
	freeSpace   uint64
}

func NewBTreeManager(totalSpace uint64) *BTreeManager {
	dm := &BTreeManager{
		treeBySize:  btree.New(32),
		treeByStart: btree.New(32),
		totalSpace:  totalSpace,
		freeSpace:   totalSpace,
	}
	block := &BTreeBlock{Start: 0, Size: totalSpace}
	dm.treeBySize.ReplaceOrInsert(BlockBySize{block})
	dm.treeByStart.ReplaceOrInsert(BlockByStart{block})
	return dm
}

func NewBTreeManagerWithBlocks(totalSpace uint64, blocks []BTreeBlock) *BTreeManager {
	dm := &BTreeManager{
		treeBySize:  btree.New(32),
		treeByStart: btree.New(32),
		totalSpace:  totalSpace,
	}

	for _, block := range blocks {
		dm.treeBySize.ReplaceOrInsert(BlockBySize{&BTreeBlock{block.Start, block.Size}})
		dm.treeByStart.ReplaceOrInsert(BlockByStart{&BTreeBlock{block.Start, block.Size}})
		dm.freeSpace += block.Size
	}
	return dm
}

func (dm *BTreeManager) Allocate(size uint64) (uint64, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.freeSpace < size {
		return 0, errors.New("NoSpaceLeft")
	}

	var bestFit *BTreeBlock
	dm.treeBySize.AscendGreaterOrEqual(BlockBySize{&BTreeBlock{Size: size}}, func(item btree.Item) bool {
		block := item.(BlockBySize).BTreeBlock
		if bestFit == nil || block.Size < bestFit.Size {
			bestFit = block
		}
		return bestFit.Size == size // 如果找到完全匹配的块，就停止搜索
	})

	if bestFit == nil {
		return 0, errors.New("NoSpaceLeft")
	}

	start := bestFit.Start
	dm.treeBySize.Delete(BlockBySize{bestFit})
	dm.treeByStart.Delete(BlockByStart{bestFit})

	if bestFit.Size > size {
		remainingBlock := &BTreeBlock{Start: start + size, Size: bestFit.Size - size}
		dm.treeBySize.ReplaceOrInsert(BlockBySize{remainingBlock})
		dm.treeByStart.ReplaceOrInsert(BlockByStart{remainingBlock})
	}

	dm.freeSpace -= size
	return start, nil
}

func (dm *BTreeManager) Free(start, size uint64) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	newBlock := &BTreeBlock{Start: start, Size: size}

	var prevBlock, nextBlock *BTreeBlock
	dm.treeByStart.DescendLessOrEqual(BlockByStart{&BTreeBlock{Start: start}}, func(item btree.Item) bool {
		prevBlock = item.(BlockByStart).BTreeBlock
		return false
	})

	dm.treeByStart.AscendGreaterOrEqual(BlockByStart{&BTreeBlock{Start: start + size}}, func(item btree.Item) bool {
		nextBlock = item.(BlockByStart).BTreeBlock
		return false
	})

	// 尝试合并与前一个块
	if prevBlock != nil && prevBlock.Start+prevBlock.Size == start {
		dm.treeBySize.Delete(BlockBySize{prevBlock})
		dm.treeByStart.Delete(BlockByStart{prevBlock})
		newBlock.Start = prevBlock.Start
		newBlock.Size += prevBlock.Size
	}

	// 尝试合并与后一个块
	if nextBlock != nil && start+size == nextBlock.Start {
		dm.treeBySize.Delete(BlockBySize{nextBlock})
		dm.treeByStart.Delete(BlockByStart{nextBlock})
		newBlock.Size += nextBlock.Size
	}

	dm.treeBySize.ReplaceOrInsert(BlockBySize{newBlock})
	dm.treeByStart.ReplaceOrInsert(BlockByStart{newBlock})
	dm.freeSpace += size
	return nil
}

func (dm *BTreeManager) GetAvailableSpace() uint64 {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.freeSpace
}
