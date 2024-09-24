# Spaceweave: 高效的磁盘空间管理系统

## 项目概述

Spaceweave 是一个高性能、并发的磁盘空间管理系统，专为高效分配和管理大规模存储空间而设计。它采用混合策略，结合了位图和 B 树数据结构，以优化空间利用率和分配速度。

## 系统架构

Spaceweave 的核心架构由以下几个主要组件构成：

1. **磁盘分配器 (DiskAllocator)**
    - 作为系统的主要接口，协调其他组件的工作。
    - 实现了空间分配、释放和利用率计算等核心功能。

2. **并发位图 (ConcurrentBitMap)**
    - 负责管理小块空间（默认 <=256KB）。
    - 使用分片技术提高并发性能。

3. **B树管理器 (BTreeManager)**
    - 处理大块空间的分配和管理（默认 >256KB）。
    - 使用两棵 B 树：一棵按大小排序，另一棵按起始地址排序。

4. **持久化管理**
    - 负责系统状态的保存和恢复，确保数据的持久性。

5. **配置管理**
    - 提供灵活的配置选项，允许用户根据具体需求调整系统参数。

## 设计方案

### 1. 混合分配策略

- 小块空间（<=256KB）使用位图管理，保证快速分配和释放。
- 大块空间（>256KB）使用 B 树管理，实现高效的空间查找和合并。

### 2. 并发优化

- 位图使用分片技术，减少锁竞争，提高并发性能。
- B 树操作采用细粒度锁，允许多个线程同时操作不同的树节点。

### 3. 空间管理

- 实现了首次适应（First-Fit）算法来分配空间。
- 支持空间的分割和合并，最大化空间利用率。

### 4. 持久化机制

- 定期将系统状态保存到磁盘，支持崩溃恢复。
- 使用 Go 的 `gob` 包进行高效的序列化和反序列化。

### 5. 可配置性

- 提供多个可配置参数，如单元大小、总空间大小、小块限制等。
- 支持自定义备份间隔和触发阈值。

## 使用方式

### 安装

```bash
go get github.com/li1213987842/spaceweave
```

### 初始化

```go
import (
    "github.com/li1213987842/spaceweave/config"
    "github.com/li1213987842/spaceweave/allocator"
)

cfg := &config.Config{
    UnitSize:             4096,           // 分配单元大小（字节）
    TotalSize:            1024 * 1024 * 1024, // 总空间大小（字节）
    SmallBlockLimit:      256 * 1024,     // 小块空间上限（字节）
    NumShards:            16,             // 位图分片数
    StatePersistencePath: "/path/to/state.gob",
    BackupIntervalSec:    300,            // 备份间隔（秒）
    BackupOperationThreshold: 1000,       // 触发备份的操作次数
}

diskAllocator, err := allocator.NewDiskAllocator(cfg)
if err != nil {
    // 处理错误
}
```

### 分配空间

```go
size := uint64(1 * 1024 * 1024) // 1MB
address, err := diskAllocator.Allocate(size)
if err != nil {
    // 处理错误
}
```

### 释放空间

```go
err := diskAllocator.Free(address, size)
if err != nil {
    // 处理错误
}
```

### 获取磁盘利用率

```go
utilization := diskAllocator.GetDiskUtilization()
```

### 关闭分配器

```go
err := diskAllocator.Close()
if err != nil {
    // 处理错误
}
```

## 性能考虑

- 小块空间的分配和释放极快，适合频繁的小规模操作。
- 大块空间的管理效率高，适合大文件存储和管理。
- 并发设计使得系统能够处理高并发的分配请求。
- 定期备份机制确保了数据的安全性，同时通过阈值控制减少了对性能的影响。

## 限制和未来改进

- 当前版本不支持动态调整总空间大小。
- 在极端情况下可能出现空间碎片化问题。
- 未来可能添加更智能的碎片整理算法。
- 计划增加更详细的监控和分析功能。

