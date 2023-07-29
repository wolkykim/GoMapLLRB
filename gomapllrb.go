// Package gomapllrb implements an in-memory key/value store using LLRB algorithm.
// LLRB (Left-Leaning Red-Black) is a self-balancing binary search tree that
// stores the keys in order which allows ordered iteration and find nearest keys.
//
// Copyright (c) 2023, Seungyoung Kim
// https://github.com/wolkykim/GoMapLLRB
package gomapllrb

import (
	"bytes"
	"fmt"
	"sync"

	"golang.org/x/exp/constraints"
)

const (
	// LLRB234 sets the tree management property and algorithm.
	LLRB234 = true // true: 2-3-4 varian(default), false: 2-3 variant
)

// Tree is the glorious tree struct.
type Tree[K constraints.Ordered] struct {
	isLess Comparator[K] // data comparator (default: string comparator)

	root  *Node[K]     // root node
	len   int          // number of object stored
	mutex sync.RWMutex // reader/writer mutual exclusion lock

	stats Stats // usage and performance metrics
}

// Node is like an apple on the apple trees.
type Node[K constraints.Ordered] struct {
	name K
	data interface{}

	red   bool
	up    *Node[K]
	left  *Node[K]
	right *Node[K]
}

// Stats provides usage statistics accessible via Stats() method.
type Stats struct {
	Put struct {
		Sum    uint64
		New    uint64
		Update uint64
	}
	Delete struct {
		Sum      uint64
		Deleted  uint64
		NotFound uint64
	}
	Get struct {
		Sum      uint64
		Found    uint64
		NotFound uint64
	}
	Perf PerfStats
}

// PerfStats are global stats for debugging purpose.
type PerfStats struct {
	Flip   uint64
	Rotate struct {
		Sum   uint64
		Left  uint64
		Right uint64
	}
}

// New creates a new tree.
func New[K constraints.Ordered]() *Tree[K] {
	return &Tree[K]{
		isLess: IsLess[K],
	}
}

// SetLess sets a user comparator function.
//
//	func myLess[K constraints.Ordered](a, b K) bool {
//	  // return true if a < b, or false
//	}
func (tree *Tree[K]) SetLess(fn Comparator[K]) {
	tree.isLess = fn
}

// Put inserts a new key or replaces old if the same key is found.
func (tree *Tree[K]) Put(name K, data interface{}) {
	tree.mutex.Lock()
	defer tree.mutex.Unlock()
	tree.root = tree.put(tree.root, name, data)
	tree.root.red = false
}

// Delete deletes the key. It returns an error if the key is not found.
func (tree *Tree[K]) Delete(name K) bool {
	tree.mutex.Lock()
	defer tree.mutex.Unlock()
	var deleted bool
	tree.root, deleted = tree.delete(tree.root, name)
	if tree.root != nil {
		tree.root.red = false
	}
	return deleted
}

// Get returns the value of the key. If key is not found, it returns Nil.
// When Nil value is expected as a actual value, use Exist() instead.
func (tree *Tree[K]) Get(name K) interface{} {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.get(tree.root, name); node != nil {
		return node.data
	}
	return nil
}

// Exist checks if the key exists.
func (tree *Tree[K]) Exist(name K) bool {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.get(tree.root, name); node != nil {
		return true
	}
	return false
}

// Min returns a min key and value.
func (tree *Tree[K]) Min() (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := findMin(tree.root); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// Max returns a max key and value.
func (tree *Tree[K]) Max() (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := findMax(tree.root); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// Bigger finds the next key bigger than given ken.
func (tree *Tree[K]) Bigger(name K) (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.bigger(tree.root, name, false); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// Smaller finds the next key bigger than given ken.
func (tree *Tree[K]) Smaller(name K) (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.smaller(tree.root, name, false); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// EqualOrBigger finds a matching key or the next bigger key.
func (tree *Tree[K]) EqualOrBigger(name K) (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.bigger(tree.root, name, true); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// EqualOrSmaller finds a matching key or the next smaller key.
func (tree *Tree[K]) EqualOrSmaller(name K) (K, interface{}, bool) {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	if node := tree.smaller(tree.root, name, true); node != nil {
		return node.name, node.data, true
	}
	var n K
	return n, nil, false
}

// Clear empties the tree without resetting the statistic metrics.
func (tree *Tree[K]) Clear() {
	tree.mutex.Lock()
	defer tree.mutex.Unlock()
	tree.root = nil
	tree.len = 0
}

// Len returns the number of object stored.
func (tree *Tree[K]) Len() int {
	return tree.len
}

// Stats returns a copy of the statistics metrics.
func (tree *Tree[K]) Stats() Stats {
	tree.stats.Put.Sum = tree.stats.Put.New + tree.stats.Put.Update
	tree.stats.Get.Sum = tree.stats.Get.Found + tree.stats.Get.NotFound
	tree.stats.Delete.Sum = tree.stats.Delete.Deleted + tree.stats.Delete.NotFound
	tree.stats.Perf = pstats
	tree.stats.Perf.Rotate.Sum = tree.stats.Perf.Rotate.Left + tree.stats.Perf.Rotate.Right
	return tree.stats
}

// ResetStats resets all the satistics metrics.
func (tree *Tree[K]) ResetStats() {
	tree.stats = Stats{}
	pstats = PerfStats{}
}

// String returns a pretty drawing of the tree structure.
//
//	┌── 6
//	|   └──[5]
//	4
//	│   ┌── 3
//	└──[2]
//	    └── 1
func (tree *Tree[K]) String() string {
	var buf bytes.Buffer
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	printNode(tree.root, &buf, nil, false)
	return buf.String()
}

// Map returns the tree in a map
func (tree *Tree[K]) Map() map[K]interface{} {
	m := make(map[K]interface{}, tree.Len())
	for it := tree.Iter(); it.Next(); {
		m[it.Key()] = it.Val()
	}
	return m
}

// String returns a statistics data in a string.
func (s Stats) String() string {
	variant := "234"
	if !LLRB234 {
		variant = "23"
	}
	numUpdate := s.Put.Sum + s.Delete.Sum
	return fmt.Sprintf("Variant:LLRB%s, Put:%d, Delete:%d, Get:%d, Rotate:%0.2f, Flip:%0.2f",
		variant, s.Put.Sum, s.Delete.Sum, s.Get.Sum,
		float64(s.Perf.Rotate.Sum)/float64(numUpdate),
		float64(s.Perf.Flip)/float64(numUpdate))
}

// Check checks that the invariants of the red-black tree are satisfied.
//
//	Root property:  The root is black.
//	Red property:   If a node is red, then both its children are black.
//	Black property: For each node, all simple paths from the node to
//	                descendant leaves contain the same number of black nodes.
//	LLRB property:  3-nodes always lean to the left and 4-nodes are balanced.
func (tree *Tree[K]) Check() error {
	if err := checkRoot(tree.root); err != nil {
		return err
	}
	if err := checkRed(tree.root); err != nil {
		return err
	}
	length := 0
	if err := checkBlack(tree.root, &length); err != nil {
		return err
	}
	return checkLLRB(tree.root)
}

/*************************************************************************
 * Iterator
 ************************************************************************/

// Iter is a iterator object.
type Iter[K constraints.Ordered] struct {
	tree *Tree[K]
	cur  *Node[K] // cursor, start from
	last *Node[K] // last node pointer after next()
	end  K        // end boundary is span is set
	span bool     // indicates the end boundary is set
	done bool     // indicates the iteration is complete
}

// Iter returns an iterator.
// Consider using IterSafe() if new key insertions or deletions are expected by
// another threads or itself during the iteration loop. In such case, the travel
// could be incomplete and could skip visiting some keys.
func (tree *Tree[K]) Iter() *Iter[K] {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	it := &Iter[K]{
		tree: tree,
		cur:  findMin(tree.root),
	}
	if it.cur == nil {
		it.done = true
	}
	return it
}

// Range returns a ranged iterator.
func (tree *Tree[K]) Range(start, end K) *Iter[K] {
	tree.mutex.RLock()
	defer tree.mutex.RUnlock()
	it := &Iter[K]{
		tree: tree,
		cur:  tree.bigger(tree.root, start, true),
		end:  end,
		span: true,
	}
	if it.cur == nil {
		it.done = true
	}
	return it
}

// Next travels the keys in the tree.
func (it *Iter[K]) Next() bool {
	if it.done {
		return false
	}
	it.last = it.cur
	it.tree.mutex.RLock()
	defer it.tree.mutex.RUnlock()
	if it.cur = it.tree.bigger(it.cur.right, it.cur.name, false); it.cur == nil {
		it.cur = it.last.up
		// go up until bigger value found
		for it.cur != nil && it.tree.isLess(it.cur.name, it.last.name) {
			it.cur = it.cur.up
		}
		// go down again
		if it.cur != nil {
			it.cur = it.tree.bigger(it.cur, it.last.name, false)
		}
		if it.cur == nil {
			it.done = true
		}
	}
	if !it.done && it.span && it.tree.isLess(it.end, it.cur.name) {
		it.done = true
	}
	return true
}

// Key returns the key name.
func (it *Iter[K]) Key() K {
	if it.last == nil {
		var k K
		return k
	}
	return it.last.name
}

// Val returns the value data.
func (it *Iter[K]) Val() interface{} {
	if it.last == nil {
		return nil
	}
	return it.last.data
}

/*************************************************************************
 * Safe Iterator
 ************************************************************************/

// IterSafe is a thread-safe iterator.
type IterSafe[K constraints.Ordered] struct {
	tree *Tree[K]
	cur  K    // cursor, start from
	last K    // copy of key after next()
	end  K    // end boundary if span is set
	span bool // indicates the end boundary is set
	done bool // indicates the iteration is complete
}

// IterSafe returns a safe iterator.
// Safe iterator isn't get affected by data insertions and deletions by other threads or itself.
// It guarantees to visit the next key with the current state of data at the time of Next() call.
// But note that this iterator is slower than Iter().
func (tree *Tree[K]) IterSafe() *IterSafe[K] {
	it := &IterSafe[K]{
		tree: tree,
	}
	if k, _, exist := tree.Min(); exist {
		it.cur = k
	} else {
		it.done = true
	}
	return it
}

// RangeSafe returns a ranged safe iterator.
func (tree *Tree[K]) RangeSafe(start, end K) *IterSafe[K] {
	it := &IterSafe[K]{
		tree: tree,
		end:  end,
		span: true,
	}
	var exist bool
	if it.cur, _, exist = it.tree.EqualOrBigger(start); !exist {
		it.done = true
	}
	return it
}

// Next travels the keys in the tree.
func (it *IterSafe[K]) Next() bool {
	if it.done {
		return false
	}
	it.last = it.cur
	var exist bool
	if it.cur, _, exist = it.tree.Bigger(it.cur); !exist {
		it.done = true
	}
	if !it.done && it.span && it.tree.isLess(it.end, it.cur) {
		it.done = true
	}
	return true
}

// Key returns the key name.
func (it *IterSafe[K]) Key() K {
	return it.last
}

// Val returns the data of the key.
func (it *IterSafe[K]) Val() interface{} {
	return it.tree.Get(it.last)
}

/*************************************************************************
 * Default comparators
 ************************************************************************/

// Comparator is the type.
type Comparator[K constraints.Ordered] func(a, b K) bool

// IsLess is the default comparator.
func IsLess[K constraints.Ordered](a, b K) bool {
	return a < b
}

/*************************************************************************
 * User data manipulation functions
 ************************************************************************/
func (tree *Tree[K]) put(node *Node[K], name K, data interface{}) *Node[K] {
	if node == nil {
		tree.len++
		tree.stats.Put.New++
		return newNode[K](name, data)
	}

	if LLRB234 {
		// split 4-nodes on the way down
		if isRed(node.left) && isRed(node.right) {
			flipColor(node)
		}
	}

	if tree.isLess(name, node.name) {
		node.left = tree.put(node.left, name, data)
		node.left.up = node
	} else if tree.isLess(node.name, name) {
		node.right = tree.put(node.right, name, data)
		node.right.up = node
	} else { // existing key found
		node.data = data
		tree.stats.Put.Update++
	}

	// fix right-leaning reds on the way up
	if isRed(node.right) && !isRed(node.left) {
		node = rotateLeft(node)
	}

	// fix two reds in a row on the way up
	if isRed(node.left) && isRed(node.left.left) {
		node = rotateRight(node)
	}

	if !LLRB234 {
		// split 4-nodes on the way up
		if isRed(node.left) && isRed(node.right) {
			flipColor(node)
		}
	}

	// return new root
	return node
}

func (tree *Tree[K]) delete(node *Node[K], name K) (*Node[K], bool) {
	if node == nil {
		tree.stats.Delete.NotFound++
		return nil, false
	}

	deleted := false
	if tree.isLess(name, node.name) {
		// move red left
		if node.left != nil && (!isRed(node.left) && !isRed(node.left.left)) {
			node = moveRedLeft(node)
		}
		// keep going down to the left
		node.left, deleted = tree.delete(node.left, name)
	} else { // right or equal
		if isRed(node.left) {
			node = rotateRight(node)
		}
		// remove if equal at the bottom
		if node.right == nil && !tree.isLess(node.name, name) {
			tree.len--
			tree.stats.Delete.Deleted++
			return nil, true
		}
		// move red right
		if node.right != nil && (!isRed(node.right) && !isRed(node.right.left)) {
			node = moveRedRight(node)
		}
		// found in the middle
		if !tree.isLess(node.name, name) {
			// we delete the min node from the right instead
			var min *Node[K]
			node.right, min = deleteMin(node.right)
			// then copy the min node to this
			node.name = min.name
			node.data = min.data
			tree.len--
			tree.stats.Delete.Deleted++
		} else {
			// keep going down to the right
			node.right, deleted = tree.delete(node.right, name)
		}
	}
	// fix right-leaning red nodes on the way up
	return fixNode(node), deleted
}

func (tree *Tree[K]) get(node *Node[K], name K) *Node[K] {
	// do linear search for performance
	for node != nil {
		if tree.isLess(name, node.name) {
			node = node.left
		} else if tree.isLess(node.name, name) {
			node = node.right
		} else {
			tree.stats.Get.Found++
			return node
		}
	}
	tree.stats.Get.NotFound++
	return nil
}

func (tree *Tree[K]) bigger(node *Node[K], name K, equal bool) *Node[K] {
	if node == nil {
		return nil
	}
	this := node
	if tree.isLess(name, node.name) {
		if node = tree.bigger(node.left, name, equal); node == nil {
			node = this
		}
	} else if tree.isLess(node.name, name) {
		node = tree.bigger(node.right, name, equal)
	} else if !equal {
		// match found, continue to the right
		node = tree.bigger(node.right, name, equal)
	}
	return node
}

func (tree *Tree[K]) smaller(node *Node[K], name K, equal bool) *Node[K] {
	if node == nil {
		return nil
	}
	this := node
	if tree.isLess(name, node.name) {
		node = tree.smaller(node.left, name, equal)
	} else if tree.isLess(node.name, name) {
		if node = tree.smaller(node.right, name, equal); node == nil {
			node = this
		}
	} else if !equal {
		// match found, continue to the left
		node = tree.smaller(node.left, name, equal)
	}
	return node
}

/*************************************************************************
 * Tree property management functions
 ************************************************************************/

var pstats PerfStats

func newNode[K constraints.Ordered](name K, data interface{}) *Node[K] {
	return &Node[K]{
		name: name,
		data: data,
		red:  true,
	}
}

func isRed[K constraints.Ordered](node *Node[K]) bool {
	if node == nil {
		return false
	}
	return node.red
}

func flipColor[K constraints.Ordered](node *Node[K]) {
	node.red = !node.red
	node.left.red = !node.left.red
	node.right.red = !node.right.red
	pstats.Flip++
}

func rotateLeft[K constraints.Ordered](node *Node[K]) *Node[K] {
	n := node.right
	n.up = node.up
	node.up = n
	node.right = n.left
	n.left = node
	n.red = n.left.red
	n.left.red = true
	pstats.Rotate.Left++
	return n
}

func rotateRight[K constraints.Ordered](node *Node[K]) *Node[K] {
	n := node.left
	n.up = node.up
	node.up = n
	node.left = n.right
	n.right = node
	n.red = n.right.red
	n.right.red = true
	pstats.Rotate.Right++
	return n
}

func moveRedLeft[K constraints.Ordered](node *Node[K]) *Node[K] {
	flipColor(node)
	if isRed(node.right.left) {
		node.right = rotateRight(node.right)
		node = rotateLeft(node)
		flipColor(node)
		if LLRB234 {
			// 2-3-4 exclusive
			if isRed(node.right.right) {
				node.right = rotateLeft(node.right)
			}
		}
	}
	return node
}

func moveRedRight[K constraints.Ordered](node *Node[K]) *Node[K] {
	flipColor(node)
	if isRed(node.left.left) {
		node = rotateRight(node)
		flipColor(node)
	}
	return node
}

func findMin[K constraints.Ordered](node *Node[K]) *Node[K] {
	if node == nil {
		return nil
	}

	for node.left != nil {
		node = node.left
	}
	return node
}

func findMax[K constraints.Ordered](node *Node[K]) *Node[K] {
	if node == nil {
		return nil
	}

	for node.right != nil {
		node = node.right
	}
	return node
}

func deleteMin[K constraints.Ordered](node *Node[K]) (*Node[K], *Node[K]) {
	if node.left == nil {
		// 3-nodes are left-leaning, so this is a leaf.
		return nil, node
	}
	if !isRed(node.left) && !isRed(node.left.left) {
		node = moveRedLeft(node)
	}
	var min *Node[K]
	node.left, min = deleteMin(node.left)
	return fixNode(node), min
}

func fixNode[K constraints.Ordered](node *Node[K]) *Node[K] {
	// rotate right red to left
	if isRed(node.right) {
		if LLRB234 {
			if isRed(node.right.left) {
				node.right = rotateRight(node.right)
			}
		}
		node = rotateLeft(node)
	}
	// rotate left red-red to right
	if isRed(node.left) && isRed(node.left.left) {
		node = rotateRight(node)
	}

	if !LLRB234 {
		// split 4-nodes
		if isRed(node.left) && isRed(node.right) {
			flipColor(node)
		}
	}
	return node
}

/*************************************************************************
 * Integrity checks
 ************************************************************************/

// checkRoot verifies that root property of the red-black tree is satisfied.
func checkRoot[K constraints.Ordered](root *Node[K]) error {
	if isRed(root) {
		return fmt.Errorf("root property violation found")
	}

	return nil
}

// checkRed verifies that red property of the red-black tree is satisfied.
func checkRed[K constraints.Ordered](node *Node[K]) error {
	if node == nil {
		return nil
	}

	if isRed(node) && (isRed(node.right) || isRed(node.left)) {
		return fmt.Errorf("red property violation found")
	}
	if err := checkRed(node.right); err != nil {
		return err
	}
	return checkRed(node.left)
}

// checkBlack verifies that black property of the red-black tree is satisfied.
func checkBlack[K constraints.Ordered](node *Node[K], length *int) error {
	if node == nil {
		*length = 1
		return nil
	}

	var rightLength int
	if err := checkBlack(node.right, &rightLength); err != nil {
		return err
	}
	var leftLength int
	if err := checkBlack(node.left, &leftLength); err != nil {
		return err
	}

	if rightLength != leftLength {
		return fmt.Errorf("black property violation found")
	}
	if !isRed(node) {
		*length = rightLength + 1
	} else {
		*length = rightLength
	}
	return nil
}

// checkLLRB verifies that LLRB property of the left-leaning red-black tree is satisfied.
func checkLLRB[K constraints.Ordered](node *Node[K]) error {
	if node == nil {
		return nil
	}

	if isRed(node.right) && !isRed(node.left) {
		return fmt.Errorf("LLRB property violation found")
	}
	if err := checkLLRB(node.right); err != nil {
		return err
	}
	return checkLLRB(node.left)
}

/*************************************************************************
 * Tree printing functions
 ************************************************************************/

type branchObj struct {
	prev *branchObj
	str  string
}

func printBranch(branch *branchObj, out *bytes.Buffer) {
	if branch == nil {
		return
	}
	printBranch(branch.prev, out)
	out.WriteString(branch.str)
}

func printNode[K constraints.Ordered](node *Node[K], out *bytes.Buffer, pbranch *branchObj, right bool) {
	if node == nil {
		return
	}

	branch := &branchObj{
		prev: pbranch,
	}

	if pbranch != nil {
		if right {
			branch.str = "    "
		} else {
			branch.str = "│   "
		}
	} else {
		branch.str = ""
	}
	printNode(node.right, out, branch, true)

	printBranch(pbranch, out)
	if pbranch != nil {
		if right {
			out.WriteString("┌──")
		} else {
			out.WriteString("└──")
		}
		if node.red {
			out.WriteString("[")
		} else {
			out.WriteString("─")
		}
	}
	out.WriteString(fmt.Sprintf("%v", node.name))
	if node.red {
		out.WriteString("]\n")
	} else {
		out.WriteString(" \n")
	}

	if pbranch != nil {
		if right {
			branch.str = "│   "
		} else {
			branch.str = "    "
		}
	} else {
		branch.str = ""
	}
	printNode(node.left, out, branch, false)
}
