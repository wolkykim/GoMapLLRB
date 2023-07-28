// Package GoMapLLRB implements an in-memory key/value store using LLRB algorithm.
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
	// GoMapLLRB supports both 2-3-4 and 2-3 variants for anyone curious.
	// This changes the characteristics of the self-balancing properties.
	LLRB234 = true // true: 2-3-4 varian(default), false: 2-3 variant
)

// Tree is the glorious tree struct
type Tree[K constraints.Ordered] struct {
	isLess Comparator[K] // data comparator (default: string comparator)

	root  *Node[K] // root node
	len   int      // number of object stored
	mutex sync.Mutex

	stats Stats // usage and performance metrics
}

// Node is like an apple on the apple trees
type Node[K constraints.Ordered] struct {
	name K
	data interface{}

	red   bool
	up    *Node[K]
	left  *Node[K]
	right *Node[K]

	tid uint8 // used in iterator
}

// Stats provides usage statistics accessible via Stats() method
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

// PerfStats are global stats for debugging purpose
type PerfStats struct {
	Flip   uint64
	Rotate struct {
		Sum   uint64
		Left  uint64
		Right uint64
	}
}

// New creates a new tree for ya! Enjoy the trees!!!
func New[K constraints.Ordered]() *Tree[K] {
	return &Tree[K]{
		isLess: isLess[K],
	}
}

// Put inserts a new key or replaces old if the same key is found
func (tree *Tree[K]) Put(name K, data interface{}) {
	tree.Lock()
	defer tree.Unlock()
	tree.root = tree.put(tree.root, name, data)
	tree.root.red = false
}

// Delete deletes the key. It returns an error if the key is not found.
func (tree *Tree[K]) Delete(name K) bool {
	tree.Lock()
	defer tree.Unlock()
	var deleted bool
	tree.root, deleted = tree.delete(tree.root, name)
	if tree.root != nil {
		tree.root.red = false
	}
	return deleted
}

// Get returns the value of the key. If key is not found, it returns Nil.
// When Nil value is expected as a actual value, use Exist() instead
func (tree *Tree[K]) Get(name K) interface{} {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.get(tree.root, name); node != nil {
		return node.data
	}
	return nil
}

// Exist checks if the key exists.
func (tree *Tree[K]) Exist(name K) bool {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.get(tree.root, name); node != nil {
		return true
	}
	return false
}

// Min returns a min key and value
func (tree *Tree[K]) Min(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := findMin(tree.root); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// Max returns a max key and value
func (tree *Tree[K]) Max(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := findMax(tree.root); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// Bigger finds the next key bigger than given ken
func (tree *Tree[K]) Bigger(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.bigger(tree.root, name, false); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// Smaller finds the next key bigger than given ken
func (tree *Tree[K]) Smaller(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.smaller(tree.root, name, false); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// EqualOrBigger finds a matching key or the next bigger key.
func (tree *Tree[K]) EqualOrBigger(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.bigger(tree.root, name, true); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// EqualOrSmaller finds a matching key or the next smaller key.
func (tree *Tree[K]) EqualOrSmaller(name K) (K, interface{}, bool) {
	tree.Lock()
	defer tree.Unlock()
	if node := tree.smaller(tree.root, name, true); node != nil {
		tree.stats.Get.Found++
		return node.name, node.data, true
	}
	tree.stats.Get.NotFound++
	var n K
	return n, nil, false
}

// Len returns the number of object stored
func (tree *Tree[K]) Len() int {
	return tree.len
}

// SetLess sets a user comparator function
//
//	func myLess[K constraints.Ordered](a, b K) bool {
//	  // return true if a < b, or false
//	}
func (tree *Tree[K]) SetLess(fn Comparator[K]) {
	tree.isLess = fn
}

// Clear empties the tree without resetting the statistic metrics
func (tree *Tree[K]) Clear() {
	tree.Lock()
	defer tree.Unlock()
	tree.root = nil
	tree.len = 0
}

// Check checks that the invariants of the red-black tree are satisfied.
//
//	Root property:  The root is black.
//	Red property:   If a node is red, then both its children are black.
//	Black property: For each node, all simple paths from the node to
//	                descendant leaves contain the same number of black nodes.
//	LLRB property:  3-nodes always lean to the left and 4-nodes are balanced.
func (tree *Tree[K]) Check() error {
	if tree == nil {
		return nil
	}

	/*
		if err := checkRoot(tree); err != nil {
			return err
		}
	*/
	if err := checkRed(tree.root); err != nil {
		return err
	}
	len := 0
	if err := checkBlack(tree.root, &len); err != nil {
		return err
	}
	if err := checkLLRB(tree.root); err != nil {
		return err
	}

	return nil
}

// Lock the tree access to the tree data structure
func (tree *Tree[K]) Lock() {
	tree.mutex.Lock()
}

// Unlock the access to the tree data structure
func (tree *Tree[K]) Unlock() {
	tree.mutex.Unlock()
}

// Stats returns a copy of the statistics metrics
func (tree *Tree[K]) Stats() Stats {
	tree.stats.Put.Sum = tree.stats.Put.New + tree.stats.Put.Update
	tree.stats.Get.Sum = tree.stats.Get.Found + tree.stats.Get.NotFound
	tree.stats.Delete.Sum = tree.stats.Delete.Deleted + tree.stats.Delete.NotFound
	tree.stats.Perf = pstats
	tree.stats.Perf.Rotate.Sum = tree.stats.Perf.Rotate.Left + tree.stats.Perf.Rotate.Right
	return tree.stats
}

// ResetStats resets all the satistics metrics
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
	tree.Lock()
	defer tree.Unlock()
	printNode(tree.root, &buf, nil, false)
	return buf.String()
}

// String returns a statistics data in a string
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

/*************************************************************************
 * Iterator
 ************************************************************************/

// Iterator
type It[K constraints.Ordered] struct {
	tree  *Tree[K]
	start K
	end   K
	span  bool
	done  bool

	name K
	data interface{}
}

// Iter returns a iterator
func (tree *Tree[K]) Iter() *It[K] {
	it := &It[K]{
		tree: tree,
	}
	if node := findMin[K](tree.root); node != nil {
		it.start = node.name
	} else {
		it.done = true
	}
	return it
}

// Range returns a ranged iterator
func (tree *Tree[K]) Range(start, end K) *It[K] {
	it := &It[K]{
		tree: tree,
		end:  end,
		span: true,
	}
	if node := tree.bigger(tree.root, start, true); node != nil {
		it.start = node.name
	} else {
		it.done = true
	}
	return it
}

// Next() iterates the tree
func (it *It[K]) Next() bool {
	if it.done {
		return false
	}

	it.name = it.start
	it.data = it.tree.Get(it.name)
	var e bool
	if it.start, _, e = it.tree.Bigger(it.start); !e {
		it.done = true
	}
	if !it.done && it.span && isLess(it.end, it.start) {
		it.done = true
	}
	return true
}

// Key returns the key name
func (it *It[K]) Key() K {
	return it.name
}

// Val returns the value data
func (it *It[K]) Val() interface{} {
	return it.data
}

/*************************************************************************
 * Default comparators
 ************************************************************************/

// Our comparator prototype
type Comparator[K constraints.Ordered] func(a, b K) bool

// Default generic comparator
func isLess[K constraints.Ordered](a, b K) bool {
	if a < b {
		return true
	}
	return false
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
// Root property:  The root is black.
func checkRoot[K constraints.Ordered](tree *Tree[K]) error {
	if tree == nil {
		return fmt.Errorf("nil tree object")
	}

	if isRed(tree.root) {
		return fmt.Errorf("root property violation found")
	}

	return nil
}

// checkRed verifies that red property of the red-black tree is satisfied.
func checkRed[K constraints.Ordered](node *Node[K]) error {
	if node == nil {
		return nil
	}

	if isRed(node) {
		if isRed(node.right) || isRed(node.left) {
			return fmt.Errorf("red property violation found")
		}
	}
	if err := checkRed(node.right); err != nil {
		return err
	}
	if err := checkRed(node.left); err != nil {
		return err
	}

	return nil
}

// checkBlack verifies that black property of the red-black tree is satisfied.
func checkBlack[K constraints.Ordered](node *Node[K], len *int) error {
	if node == nil {
		*len = 1
		return nil
	}

	var rightLen int
	if err := checkBlack(node.right, &rightLen); err != nil {
		return err
	}
	var leftLen int
	if err := checkBlack(node.left, &leftLen); err != nil {
		return err
	}

	if rightLen != leftLen {
		return fmt.Errorf("black property violation found")
	}
	if !isRed(node) {
		*len = rightLen + 1
	} else {
		*len = rightLen
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
	if err := checkLLRB(node.left); err != nil {
		return err
	}

	return nil
}

/*************************************************************************
 * Additional features
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
