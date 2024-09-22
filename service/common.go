package service

import (
	"github.com/li1213987842/spaceweave/config"
	"github.com/li1213987842/spaceweave/internal/allocator"
)

var (
	ServConfig     *config.Config
	AllocatorStore allocator.DiskAllocator
)
