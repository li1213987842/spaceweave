package allocator

import (
	"math/bits"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

type ConcurrentBitMap struct {
	shards []Shard
}

type Shard struct {
	bits []uint64
	mu   sync.RWMutex
}

func NewBitMap(size, shards uint64) *ConcurrentBitMap {
	// Initialize bitmap shards
	bitmapSize := size / 64
	shardSize := bitmapSize / shards

	bm := &ConcurrentBitMap{
		shards: make([]Shard, shards),
	}
	for i := range bm.shards {
		bm.shards[i].bits = make([]uint64, shardSize)
	}
	return bm
}

func (b *ConcurrentBitMap) Allocate(size uint64) (uint64, error) {
	shardCount := uint64(len(b.shards))
	startShard := uint64(uint32(random.Int63())) % shardCount

	for i := uint64(0); i < shardCount; i++ {
		shardIndex := (startShard + i) % shardCount
		shard := &b.shards[shardIndex]

		shard.mu.Lock()
		start, ok := allocateInShard(shard.bits, size)
		shard.mu.Unlock()
		if ok {
			return shardIndex*uint64(len(shard.bits))*64 + start, nil
		}
	}
	return 0, ErrNoSpaceLeft
}

func allocateInShard(bits []uint64, size uint64) (uint64, bool) {
	if size == 0 {
		return 0, false
	}

	consecutiveFree := uint64(0)
	start := uint64(0)

	for i, block := range bits {
		for j := 0; j < 64; j++ {
			if block&(1<<j) == 0 {
				if consecutiveFree == 0 {
					start = uint64(i*64 + j)
				}
				consecutiveFree++
				if consecutiveFree == size {
					// 找到足够的连续空间，标记为已分配
					markAllocated(bits, start, size)
					return start, true
				}
			} else {
				consecutiveFree = 0
			}
		}
	}

	return 0, false
}

func markAllocated(bits []uint64, start, size uint64) {
	for i := start; i < start+size; i++ {
		blockIndex := i / 64
		bitIndex := i % 64
		bits[blockIndex] |= 1 << bitIndex
	}
}

func (b *ConcurrentBitMap) Free(start, size uint64) error {
	shardIndex := start / uint64(len(b.shards[0].bits)*64)
	bitStart := start % uint64(len(b.shards[0].bits)*64)

	shard := &b.shards[shardIndex]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	for size > 0 {
		bitIndex := bitStart / 64
		bitOffset := bitStart % 64
		bitsToFree := uint64(64) - bitOffset
		if bitsToFree > size {
			bitsToFree = size
		}

		mask := ((uint64(1) << bitsToFree) - 1) << bitOffset
		shard.bits[bitIndex] &= ^mask

		size -= bitsToFree
		bitStart += bitsToFree
	}

	return nil
}

func (b *ConcurrentBitMap) freeSmallInShard(shardIndex, fromBit, toBit uint64) {
	shard := &b.shards[shardIndex]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	startIndex := fromBit / 64
	endIndex := toBit / 64

	for i := startIndex; i <= endIndex; i++ {
		var mask uint64
		if i == startIndex {
			if i == endIndex {
				mask = (^uint64(0) >> (63 - (toBit % 64))) & (^uint64(0) << (fromBit % 64))
			} else {
				mask = ^uint64(0) << (fromBit % 64)
			}
		} else if i == endIndex {
			mask = ^uint64(0) >> (63 - (toBit % 64))
		} else {
			mask = ^uint64(0)
		}

		shard.bits[i] &= ^mask
	}
}

func (b *ConcurrentBitMap) GetAvailableSpace() uint64 {
	var totalUnused uint64
	var wg sync.WaitGroup
	for i := range b.shards {
		wg.Add(1)
		go func(shard *Shard) {
			defer wg.Done()
			shard.mu.RLock()
			defer shard.mu.RUnlock()

			var unused uint64
			for _, v := range shard.bits {
				unused += uint64(64 - bits.OnesCount64(v))
			}
			atomic.AddUint64(&totalUnused, unused)
		}(&b.shards[i])
	}
	wg.Wait()

	return totalUnused
}
