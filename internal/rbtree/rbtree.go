package rbtree

import (
	"errors"
	"sync"
)

type Color bool

const (
	RED   Color = true
	BLACK Color = false
)

type Node struct {
	start, size uint64
	color       Color
	left, right *Node
	parent      *Node
}

type RBTree struct {
	root       *Node
	nil        *Node // Sentinel node
	mu         sync.RWMutex
	totalSpace uint64
}

func NewRBTree(start, totalUnits uint64) *RBTree {
	nil := &Node{color: BLACK}
	tree := &RBTree{nil: nil, root: nil, totalSpace: totalUnits}
	tree.insert(start, totalUnits)
	return tree
}

func (t *RBTree) Allocate(size uint64) (uint64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	node := t.findBestFit(size)
	if node == nil || node == t.nil {
		return 0, errors.New("NoSpaceLeft") // Allocation failed
	}

	start := node.start
	if node.size > size {
		// Split the node
		newNode := &Node{
			start: start + size,
			size:  node.size - size,
			color: RED, // New nodes are always red
		}
		t.insertNode(newNode)
		node.size = size
	}
	t.delete(node)
	return start, nil
}

func (t *RBTree) Free(start, size uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	node := &Node{start: start, size: size}
	prev := t.findLessThan(start)
	next := t.findGreaterThan(start)

	// 尝试与前一个节点合并
	if prev != nil && prev.start+prev.size == start {
		prev.size += size
		node = prev
	} else {
		t.insert(start, size)
		node = t.findEqual(start)
	}

	// 尝试与后一个节点合并
	if next != nil && node.start+node.size == next.start {
		node.size += next.size
		t.delete(next)
	}

	return nil
}

func (t *RBTree) findEqual(start uint64) *Node {
	current := t.root
	for current != t.nil {
		if current.start == start {
			return current
		} else if start < current.start {
			current = current.left
		} else {
			current = current.right
		}
	}
	return nil
}

func (t *RBTree) insert(start, size uint64) {
	node := &Node{start: start, size: size, color: RED}
	y := t.nil
	x := t.root

	for x != t.nil {
		y = x
		if node.start < x.start {
			x = x.left
		} else {
			x = x.right
		}
	}

	node.parent = y
	if y == t.nil {
		t.root = node
	} else if node.start < y.start {
		y.left = node
	} else {
		y.right = node
	}

	node.left = t.nil
	node.right = t.nil
	t.insertFixup(node)
}

func (t *RBTree) insertNode(node *Node) {
	y := t.nil
	x := t.root

	for x != t.nil {
		y = x
		if node.start < x.start {
			x = x.left
		} else {
			x = x.right
		}
	}

	node.parent = y
	if y == t.nil {
		t.root = node
	} else if node.start < y.start {
		y.left = node
	} else {
		y.right = node
	}

	node.left = t.nil
	node.right = t.nil
	node.color = RED
	t.insertFixup(node)
}

func (t *RBTree) insertFixup(z *Node) {
	for z.parent.color == RED {
		if z.parent == z.parent.parent.left {
			y := z.parent.parent.right
			if y.color == RED {
				z.parent.color = BLACK
				y.color = BLACK
				z.parent.parent.color = RED
				z = z.parent.parent
			} else {
				if z == z.parent.right {
					z = z.parent
					t.leftRotate(z)
				}
				z.parent.color = BLACK
				z.parent.parent.color = RED
				t.rightRotate(z.parent.parent)
			}
		} else {
			y := z.parent.parent.left
			if y.color == RED {
				z.parent.color = BLACK
				y.color = BLACK
				z.parent.parent.color = RED
				z = z.parent.parent
			} else {
				if z == z.parent.left {
					z = z.parent
					t.rightRotate(z)
				}
				z.parent.color = BLACK
				z.parent.parent.color = RED
				t.leftRotate(z.parent.parent)
			}
		}
	}
	t.root.color = BLACK
}

func (t *RBTree) leftRotate(x *Node) {
	y := x.right
	x.right = y.left
	if y.left != t.nil {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == t.nil {
		t.root = y
	} else if x == x.parent.left {
		x.parent.left = y
	} else {
		x.parent.right = y
	}
	y.left = x
	x.parent = y
}

func (t *RBTree) rightRotate(y *Node) {
	x := y.left
	y.left = x.right
	if x.right != t.nil {
		x.right.parent = y
	}
	x.parent = y.parent
	if y.parent == t.nil {
		t.root = x
	} else if y == y.parent.right {
		y.parent.right = x
	} else {
		y.parent.left = x
	}
	x.right = y
	y.parent = x
}

func (t *RBTree) delete(z *Node) error {
	if z == nil || z == t.nil {
		return errors.New("node to delete is nil or sentinel")
	}

	y := z
	yOriginalColor := y.color
	var x *Node

	if z.left == t.nil {
		x = z.right
		t.transplant(z, z.right)
	} else if z.right == t.nil {
		x = z.left
		t.transplant(z, z.left)
	} else {
		y = t.minimum(z.right)
		yOriginalColor = y.color
		x = y.right
		if y.parent == z {
			x.parent = y
		} else {
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.color = z.color
	}

	if yOriginalColor == BLACK {
		t.deleteFixup(x)
	}

	return nil
}

func (t *RBTree) deleteFixup(x *Node) {
	for x != t.root && (x == t.nil || x.color == BLACK) {
		if x == x.parent.left {
			w := x.parent.right
			if w.color == RED {
				w.color = BLACK
				x.parent.color = RED
				t.leftRotate(x.parent)
				w = x.parent.right
			}
			if (w.left == t.nil || w.left.color == BLACK) &&
				(w.right == t.nil || w.right.color == BLACK) {
				w.color = RED
				x = x.parent
			} else {
				if w.right == t.nil || w.right.color == BLACK {
					if w.left != t.nil {
						w.left.color = BLACK
					}
					w.color = RED
					t.rightRotate(w)
					w = x.parent.right
				}
				w.color = x.parent.color
				x.parent.color = BLACK
				if w.right != t.nil {
					w.right.color = BLACK
				}
				t.leftRotate(x.parent)
				x = t.root
			}
		} else {
			// Mirror image of the above case
			w := x.parent.left
			if w.color == RED {
				w.color = BLACK
				x.parent.color = RED
				t.rightRotate(x.parent)
				w = x.parent.left
			}
			if (w.right == t.nil || w.right.color == BLACK) &&
				(w.left == t.nil || w.left.color == BLACK) {
				w.color = RED
				x = x.parent
			} else {
				if w.left == t.nil || w.left.color == BLACK {
					if w.right != t.nil {
						w.right.color = BLACK
					}
					w.color = RED
					t.leftRotate(w)
					w = x.parent.left
				}
				w.color = x.parent.color
				x.parent.color = BLACK
				if w.left != t.nil {
					w.left.color = BLACK
				}
				t.rightRotate(x.parent)
				x = t.root
			}
		}
	}
	if x != t.nil {
		x.color = BLACK
	}
}

func (t *RBTree) transplant(u, v *Node) {
	if u.parent == t.nil {
		t.root = v
	} else if u == u.parent.left {
		u.parent.left = v
	} else {
		u.parent.right = v
	}
	v.parent = u.parent
}

func (t *RBTree) minimum(x *Node) *Node {
	if x == nil {
		return t.nil
	}
	for x != t.nil && x.left != t.nil {
		x = x.left
	}
	return x
}

func (t *RBTree) findBestFit(size uint64) *Node {
	best := t.nil
	current := t.root

	for current != t.nil {
		if current.size >= size {
			if best == t.nil || current.size < best.size {
				best = current
			}
			// 继续查找左子树，可能有更小但足够大的块
			current = current.left
		} else {
			current = current.right
		}
	}

	// 如果找到的最佳块比请求的大小大很多，考虑分割
	if best != t.nil && best.size > size*2 {
		return t.splitNode(best, size)
	}

	return best
}

func (t *RBTree) splitNode(node *Node, size uint64) *Node {
	if node.size <= size {
		return node
	}

	newNode := &Node{
		start: node.start + size,
		size:  node.size - size,
		color: RED,
	}

	node.size = size
	t.insertNode(newNode)

	return node
}

func (t *RBTree) findLessThan(start uint64) *Node {
	var result *Node
	current := t.root

	for current != t.nil {
		if current.start < start {
			result = current
			current = current.right
		} else {
			current = current.left
		}
	}

	return result
}

func (t *RBTree) findGreaterThan(start uint64) *Node {
	var result *Node
	current := t.root

	for current != t.nil {
		if current.start > start {
			result = current
			current = current.left
		} else {
			current = current.right
		}
	}

	return result
}

func (t *RBTree) GetAvailableSpace() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getAvailableSpaceRecursive(t.root)
}

func (t *RBTree) getAvailableSpaceRecursive(node *Node) uint64 {
	if node == t.nil {
		return 0
	}

	leftSpace := t.getAvailableSpaceRecursive(node.left)
	rightSpace := t.getAvailableSpaceRecursive(node.right)
	return node.size + leftSpace + rightSpace
}

func (t *RBTree) GetUtilization() float64 {
	availableSpace := t.GetAvailableSpace()
	return 1 - float64(availableSpace)/float64(t.totalSpace)
}

func (t *RBTree) Defragment() {
	t.mu.Lock()
	defer t.mu.Unlock()

	nodes := t.inorderTraversal()
	mergedNodes := t.mergeAdjacentNodes(nodes)

	// 进一步合并小块
	finalNodes := t.mergeSmallBlocks(mergedNodes)

	t.rebuildTree(finalNodes)
}

func (t *RBTree) mergeSmallBlocks(nodes []*Node) []*Node {
	if len(nodes) < 2 {
		return nodes
	}

	var result []*Node
	current := nodes[0]

	for i := 1; i < len(nodes); i++ {
		if current.size < t.totalSpace/100 && nodes[i].size < t.totalSpace/100 {
			// 合并小块
			current.size += nodes[i].size
		} else {
			result = append(result, current)
			current = nodes[i]
		}
	}
	result = append(result, current)

	return result
}

func (t *RBTree) inorderTraversal() []*Node {
	var nodes []*Node
	t.inorderTraversalRecursive(t.root, &nodes)
	return nodes
}

func (t *RBTree) inorderTraversalRecursive(node *Node, nodes *[]*Node) {
	if node == t.nil {
		return
	}
	t.inorderTraversalRecursive(node.left, nodes)
	*nodes = append(*nodes, node)
	t.inorderTraversalRecursive(node.right, nodes)
}

func (t *RBTree) mergeAdjacentNodes(nodes []*Node) []*Node {
	if len(nodes) < 2 {
		return nodes
	}

	var mergedNodes []*Node
	current := nodes[0]

	for i := 1; i < len(nodes); i++ {
		if current.start+current.size == nodes[i].start {
			// 合并相邻节点
			current.size += nodes[i].size
		} else {
			mergedNodes = append(mergedNodes, current)
			current = nodes[i]
		}
	}
	mergedNodes = append(mergedNodes, current)

	return mergedNodes
}

func (t *RBTree) rebuildTree(nodes []*Node) {
	t.root = t.nil
	for _, node := range nodes {
		node.left = t.nil
		node.right = t.nil
		node.parent = t.nil
		node.color = RED
		t.insertNode(node)
	}
	t.root.color = BLACK
}
