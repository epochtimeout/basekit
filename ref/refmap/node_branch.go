package refmap

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/basecomplextech/baselibrary/collect/slices"
	"github.com/basecomplextech/baselibrary/ref"
)

var _ node[any, ref.Ref] = (*branchNode[any, ref.Ref])(nil)

type branchNode[K any, V ref.Ref] struct {
	items []branchItem[K, V]

	mut  bool
	refs int64
}

type branchItem[K any, V ref.Ref] struct {
	minKey K
	node   node[K, V]
}

// newBranchNode returns a new mutable node, moves the children to it.
func newBranchNode[K any, V ref.Ref](children ...node[K, V]) *branchNode[K, V] {
	// Make node
	n := acquireBranch[K, V]()
	n.init(len(children))
	n.mut = true
	n.refs = 1

	// Move children, do not retain them
	for _, child := range children {
		item := branchItem[K, V]{
			minKey: child.minKey(),
			node:   child,
		}
		n.items = append(n.items, item)
	}
	return n
}

// cloneBranchNode returns a mutable node clone, retains its children.
func cloneBranchNode[K any, V ref.Ref](n *branchNode[K, V]) *branchNode[K, V] {
	// Copy node
	n1 := acquireBranch[K, V]()
	n1.init(cap(n.items))
	n1.mut = true
	n1.refs = 1

	n1.items = n1.items[:len(n.items)]
	copy(n1.items, n.items)

	// Retain children
	for _, child := range n1.items {
		child.node.retain()
	}
	return n1
}

// nextBranchNode returns a new mutable node on a split, moves items to it.
func nextBranchNode[K any, V ref.Ref](items []branchItem[K, V]) *branchNode[K, V] {
	// Make node
	n := acquireBranch[K, V]()
	n.init(cap(items))
	n.mut = true
	n.refs = 1

	n.items = n.items[:len(items)]
	copy(n.items, items)

	// No need to retain children
	// We have moved them to the new node
	return n
}

// reset

func (s *branchNode[K, V]) init(n int) {
	if cap(s.items) < n {
		s.items = make([]branchItem[K, V], 0, n)
	}
}

func (s *branchNode[K, V]) reset() {
	items := slices.Clear(s.items)

	*s = branchNode[K, V]{}
	s.items = items
}

// retain/release

func (n *branchNode[K, V]) retain() {
	v := atomic.AddInt64(&n.refs, 1)
	if v == 1 {
		panic("retained already released node")
	}
}

func (n *branchNode[K, V]) release() {
	v := atomic.AddInt64(&n.refs, -1)
	if v < 0 {
		panic("released already released node")
	}
	if v > 0 {
		return
	}

	// Release children
	for _, item := range n.items {
		item.node.release()
	}
	n.items = slices.Clear(n.items)

	// Release node
	releaseBranch[K, V](n)
}

func (n *branchNode[K, V]) refcount() int64 {
	return n.refs
}

// attrs

func (n *branchNode[K, V]) length() int {
	return len(n.items)
}

func (n *branchNode[K, V]) minKey() K {
	return n.items[0].minKey
}

func (n *branchNode[K, V]) maxKey() K {
	last := n.items[len(n.items)-1]
	return last.node.maxKey()
}

// mutate

func (n *branchNode[K, V]) clone() node[K, V] {
	return cloneBranchNode(n)
}

func (n *branchNode[K, V]) freeze() {
	if !n.mut {
		return
	}

	for _, child := range n.items {
		child.node.freeze()
	}

	n.mut = false
}

func (n *branchNode[K, V]) mutable() bool {
	return n.mut
}

// methods

// indexOf returns a child node index which range contains a key.
// indexOf finds the first node after a key and return  theprevious node.
func (n *branchNode[K, V]) indexOf(key K, compare CompareFunc[K]) int {
	index := sort.Search(len(n.items), func(i int) bool {
		minKey := n.items[i].minKey
		cmp := compare(minKey, key)
		return cmp > 0
	})
	if index > 0 {
		return index - 1
	}
	return 0
}

func (n *branchNode[K, V]) get(key K, compare CompareFunc[K]) (V, bool) {
	index := n.indexOf(key, compare)
	node := n.child(index)
	return node.get(key, compare)
}

func (n *branchNode[K, V]) put(key K, value V, compare CompareFunc[K]) bool {
	if !n.mut {
		panic("operation on immutable node")
	}

	// Find child node with key
	index := n.indexOf(key, compare)
	node := n.mutateChild(index)

	// Split node if full
	if node.length() >= maxItems {
		n.splitChild(index)

		index = n.indexOf(key, compare)
		node = n.mutateChild(index)
	}

	// Insert item
	mod := node.put(key, value, compare)

	// Update min key
	n.items[index].minKey = node.minKey()
	return mod
}

func (n *branchNode[K, V]) delete(key K, compare CompareFunc[K]) bool {
	if !n.mut {
		panic("operation on immutable node")
	}

	// Find child node with key
	index := n.indexOf(key, compare)
	node := n.mutateChild(index)

	// Delete item
	mod := node.delete(key, compare)
	if !mod {
		return false
	}

	// Delete child if empty
	if node.length() == 0 {
		n.deleteChild(index)
		return true
	}

	// Update min key
	n.items[index].minKey = node.minKey()
	return true
}

func (n *branchNode[K, V]) contains(key K, compare CompareFunc[K]) bool {
	index := n.indexOf(key, compare)
	if index >= len(n.items) {
		return false
	}

	node := n.child(index)
	return node.contains(key, compare)
}

func (n *branchNode[K, V]) split() (node[K, V], bool) {
	if !n.mut {
		panic("operation on immutable node")
	}

	if len(n.items) < maxItems {
		return nil, false
	}

	// Calc middle index
	middle := len(n.items) / 2

	// Move items to next node
	next := nextBranchNode(n.items[middle:])

	// Clear and truncate moved items,
	// Do not release them, we have moved them to the new node
	for i := middle; i < len(n.items); i++ {
		n.items[i] = branchItem[K, V]{}
	}
	n.items = n.items[:middle]
	return next, true
}

// children

func (n *branchNode[K, V]) child(index int) node[K, V] {
	if index >= len(n.items) {
		panic("index out of range")
	}

	child := n.items[index]
	return child.node
}

func (n *branchNode[K, V]) deleteChild(index int) {
	if index >= len(n.items) {
		panic("index out of range")
	}

	// Release node
	node := n.items[index].node
	node.release()

	// Shift items left
	copy(n.items[index:], n.items[index+1:])

	// Truncate items
	n.items[len(n.items)-1] = branchItem[K, V]{}
	n.items = n.items[:len(n.items)-1]
}

func (n *branchNode[K, V]) mutateChild(index int) node[K, V] {
	if !n.mut {
		panic("operation on immutable node")
	}

	// Return if mutable
	node := n.child(index)
	if node.mutable() {
		return node
	}

	// Clone and replace node
	prev := node
	mut := node.clone()
	n.items[index].node = mut

	// Release previous node
	prev.release()
	return mut
}

func (n *branchNode[K, V]) splitChild(index int) bool {
	if !n.mut {
		panic("operation on immutable node")
	}

	// Maybe split child
	node := n.child(index)
	next, ok := node.split()
	if !ok {
		return false
	}

	// Grow capacity
	if cap(n.items) < len(n.items)+1 {
		new := 2*len(n.items) + 1
		items := make([]branchItem[K, V], len(n.items), new)

		copy(items, n.items)
		n.items = items
	}

	// Shift items right
	n.items = n.items[:len(n.items)+1]
	copy(n.items[index+2:], n.items[index+1:])

	// Update min key
	n.items[index].minKey = node.minKey()

	// Insert new node
	n.items[index+1] = branchItem[K, V]{
		node:   next,
		minKey: next.minKey(),
	}
	return true
}

// branch state pool

var branchPools = &sync.Map{}

func acquireBranch[K any, V ref.Ref]() *branchNode[K, V] {
	pool := getBranchPool[K, V]()

	v := pool.Get()
	if v != nil {
		return v.(*branchNode[K, V])
	}
	return &branchNode[K, V]{}
}

func releaseBranch[K any, V ref.Ref](s *branchNode[K, V]) {
	s.reset()

	pool := getBranchPool[K, V]()
	pool.Put(s)
}

func getBranchPool[K any, V ref.Ref]() *sync.Pool {
	var key poolKey[K, V]

	p, ok := branchPools.Load(key)
	if ok {
		return p.(*sync.Pool)
	}

	p, _ = branchPools.LoadOrStore(key, &sync.Pool{})
	return p.(*sync.Pool)
}
