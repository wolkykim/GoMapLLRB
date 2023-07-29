package gomapllrb

import (
	"encoding/binary"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/spaolacci/murmur3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/constraints"
)

const (
	VERBOSE = false // enable visual inspections
)

// Test growth of tree
// Example taken from the inventor's presentation slide p24-p25.
// https://sedgewick.io/wp-content/uploads/2022/03/2008-09LLRB.pdf
//
// Key insertion sequence : A S E R C D I N B X
// 2-3-4 LLRB structure must form as below.
//
//	    ┌── X
//	    │   └──[S]
//	┌── R
//	│   └── N
//	│       └──[I]
//	E
//	│   ┌── D
//	└── C
//	    └── B
//	        └──[A]
//
// The nodes A, I and S are Red. Others are Black.
func TestGrowh(t *testing.T) {
	title("Test growth of tree / A S E R C D I N B X")
	assert := assert.New(t)

	keys := []string{"A", "S", "E", "R", "C", "D", "I", "N", "B", "X"}
	tree := New[string]()

	for _, k := range keys {
		fmt.Printf("Put key: %s\n", k)
		tree.Put(k, "")
		assertTreeCheck(t, tree, true)
	}

	assert.Equal("E", tree.root.name)
	assert.Equal(false, tree.root.red)
	assert.Equal("C", tree.root.left.name)
	assert.Equal(false, tree.root.left.red)
	assert.Equal("R", tree.root.right.name)
	assert.Equal(false, tree.root.right.red)
	assert.Equal("B", tree.root.left.left.name)
	assert.Equal(false, tree.root.left.left.red)
	assert.Equal("D", tree.root.left.right.name)
	assert.Equal(false, tree.root.left.right.red)
	assert.Equal("N", tree.root.right.left.name)
	assert.Equal(false, tree.root.right.left.red)
	assert.Equal("X", tree.root.right.right.name)
	assert.Equal(false, tree.root.right.right.red)
	assert.Equal("A", tree.root.left.left.left.name)
	assert.Equal(true, tree.root.left.left.left.red)
	assert.Equal("I", tree.root.right.left.left.name)
	assert.Equal(true, tree.root.right.left.left.red)
	assert.Equal("S", tree.root.right.right.left.name)
	assert.Equal(true, tree.root.right.right.left.red)

	for _, k := range keys {
		tree.Delete(k)
		assertTreeCheck(t, tree, false)
	}
}

func TestGrowhVisualInspection(t *testing.T) {
	if !VERBOSE {
		return
	}
	title("Visual inspection")

	tree := New[int]()
	for i := 0; i < 100; i++ {
		tree.Put(int(hash32(i)%1000), nil)
	}
	assertTreeCheck(t, tree, true)

	for it := tree.Iter(); it.Next(); {
		fmt.Printf("%d ", it.Key())
	}
	fmt.Println()
	for it := tree.IterSafe(); it.Next(); {
		fmt.Printf("%d ", it.Key())
	}
	fmt.Println()
}

func TestBasics(t *testing.T) {
	title("Test basics")
	assert := assert.New(t)

	keys := []int{10, 20, 30, 40, 50, 60, 70, 80}
	tree := New[int]()
	assert.Equal(0, tree.Len())

	// insert
	for _, k := range keys {
		tree.Put(k, k)
		assertTreeCheck(t, tree, false)
	}
	assert.Equal(len(keys), tree.Len())
	assertTreeCheck(t, tree, true)

	// verify
	for _, k := range keys {
		assert.Equal(k, tree.Get(k))
	}

	// not found case
	assert.Equal(nil, tree.Get(0))

	// delete
	for _, k := range keys {
		assert.NotNil(tree.Delete(k))
		assertTreeCheck(t, tree, false)
	}
	assert.Equal(0, tree.Len())

	// overwrite
	tree.Put(1, 1)
	assert.Equal(1, tree.Get(1))
	assert.Equal(1, tree.Len())
	tree.Put(1, 10)
	assert.Equal(10, tree.Get(1))
	assert.Equal(1, tree.Len())

	// clear
	tree.Clear()
	assert.Equal(0, tree.Len())
}

func TestGetters(t *testing.T) {
	title("Test Getters")
	assert := assert.New(t)

	keys := []int{10, 20, 30, 40, 50, 60, 70, 80}
	tree := New[int]()

	// test empty table
	_, _, e := tree.Min()
	assert.False(e)
	_, _, e = tree.Max()
	assert.False(e)

	// insert
	for _, k := range keys {
		tree.Put(k, k)
	}

	// test Exist()
	assert.True(tree.Exist(10))
	assert.False(tree.Exist(0))

	// test Min/Max
	min, _, _ := tree.Min()
	assert.Equal(10, min)
	max, _, _ := tree.Max()
	assert.Equal(80, max)

	// test Bigger
	for i := 0; i < len(keys)-1; i++ {
		k, _, e := tree.Bigger((i + 1) * 10)
		assert.True(e)
		assert.Equal((i+2)*10, k)
	}
	_, _, e = tree.Bigger(80)
	assert.False(e)

	// test Smaller
	_, _, e = tree.Smaller(10)
	assert.False(e)
	for i := 1; i < len(keys); i++ {
		k, _, e := tree.Smaller((i + 1) * 10)
		assert.True(e)
		assert.Equal(i*10, k)
	}

	// test EqualOrBigger & EqualOrSmaller
	for _, k := range keys {
		n, v, e := tree.EqualOrBigger(k)
		assert.True(e)
		assert.Equal(k, n)
		assert.Equal(k, v)
		n, v, e = tree.EqualOrSmaller(k)
		assert.True(e)
		assert.Equal(k, n)
		assert.Equal(k, v)
	}

	// test EqualOrBigger
	for i := range keys {
		k, _, e := tree.EqualOrBigger(i*10 + 5)
		assert.True(e)
		assert.Equal((i+1)*10, k)
	}
	_, _, e = tree.EqualOrBigger(90)
	assert.False(e)

	// test EqualOrSmaller
	_, _, e = tree.EqualOrSmaller(5)
	assert.False(e)
	for i := range keys {
		k, _, e := tree.EqualOrSmaller((i+1)*10 + 5)
		assert.True(e)
		assert.Equal((i+1)*10, k)
	}
}

func TestIter(t *testing.T) {
	title("Test Iter()")
	assert := assert.New(t)
	tree := New[int]()

	// test with empty table
	it := tree.Iter()
	assert.False(it.Next())
	assert.Equal(0, it.Key())
	assert.Nil(it.Val())
	it = tree.Range(0, 0)
	assert.False(it.Next())

	// insert
	for _, k := range []int{7, 1, 3, 9, 5} {
		tree.Put(k, nil)
	}

	it = tree.Iter()
	assert.True(it.Next())
	assert.Equal(1, it.Key())
	assert.True(it.Next())
	assert.Equal(3, it.Key())
	assert.True(it.Next())
	assert.Equal(5, it.Key())
	assert.True(it.Next())
	assert.Equal(7, it.Key())
	assert.True(it.Next())
	assert.Equal(9, it.Key())
	assert.False(it.Next())

	it = tree.Range(3, 8)
	assert.True(it.Next())
	assert.Equal(3, it.Key())
	assert.True(it.Next())
	assert.Equal(5, it.Key())
	assert.True(it.Next())
	assert.Equal(7, it.Key())
	assert.False(it.Next())
}

func TestIterSafe(t *testing.T) {
	title("Test IterSafe")
	assert := assert.New(t)
	tree := New[int]()

	// test with empty table
	it := tree.IterSafe()
	assert.False(it.Next())
	it = tree.RangeSafe(0, 0)
	assert.False(it.Next())

	// insert
	for _, k := range []int{7, 1, 3, 9, 5} {
		tree.Put(k, nil)
	}

	it = tree.IterSafe()
	assert.True(it.Next())
	assert.Equal(1, it.Key())
	assert.True(it.Next())
	assert.Equal(3, it.Key())
	assert.True(it.Next())
	assert.Equal(5, it.Key())
	assert.True(it.Next())
	assert.Equal(7, it.Key())
	assert.True(it.Next())
	assert.Equal(9, it.Key())
	assert.False(it.Next())
	assert.False(it.Next())

	it = tree.RangeSafe(3, 8)
	assert.True(it.Next())
	assert.Equal(3, it.Key())
	assert.True(it.Next())
	assert.Equal(5, it.Key())
	assert.True(it.Next())
	assert.Equal(7, it.Key())
	assert.False(it.Next())
}

func TestMap(t *testing.T) {
	title("Test Map()")
	assert := assert.New(t)

	tree := New[int]()
	for _, k := range []int{7, 1, 3, 9, 5} {
		tree.Put(k, k)
	}
	m := tree.Map()
	assert.Equal(tree.Len(), len(m))
	for _, k := range []int{7, 1, 3, 9, 5} {
		assert.Equal(k, m[k])
	}
}

func TestPerformanceRandom(t *testing.T) {
	title("Test perfmance / random")
	num := 1000000
	keys := make([]uint32, num, num)
	for i := 0; i < num; i++ {
		keys[i] = hash32(i)
	}
	perfTest(t, keys)
}

func TestPerformanceAscending(t *testing.T) {
	title("Test perfmance / ascending")
	num := 1000000
	keys := make([]uint32, num, num)
	for i := 0; i < num; i++ {
		keys[i] = uint32(i)
	}
	perfTest(t, keys)
}

/*************************************************************************
 * Helpers
 ************************************************************************/

func title(str string) {
	fmt.Printf("* TEST : %s\n", str)
}

func assertTreeCheck[K constraints.Ordered](t *testing.T, tree *Tree[K], verbose bool) {
	if err := tree.Check(); err != nil {
		t.Error(err)
		log.Println("ERROR:", err)
		verbose = true
	}
	if verbose {
		fmt.Print(tree)
		fmt.Printf("(#nodes %d, #red %d, #black %d\n", tree.Len(), 0, 0)
	}
}

func hash32(num int) uint32 {
	numBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(numBytes, uint32(num))
	hash := murmur3.New32()
	hash.Write(numBytes)
	return hash.Sum32()
}

func perfTest(t *testing.T, keys []uint32) {
	assert := assert.New(t)
	tree := New[uint32]()

	// print key samples
	fmt.Printf("  Sample")
	for i, k := range keys {
		fmt.Printf(" %v", k)
		if i == 9 {
			break
		}
	}
	fmt.Printf(", ... (Total %d)\n", len(keys))

	// put
	start := time.Now()
	for i, k := range keys {
		tree.Put(k, nil)
		if VERBOSE && i == 50 {
			assertTreeCheck(t, tree, true)
		}
	}
	stats := tree.Stats()
	fmt.Printf("  Put %d keys:\t\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)
	assert.Less(0, tree.Len())
	assertTreeCheck(t, tree, false)

	// find
	tree.ResetStats()
	start = time.Now()
	for _, k := range keys {
		tree.Exist(k)
	}
	stats = tree.Stats()
	fmt.Printf("  Find %d keys:\t\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)

	// iterator
	tree.ResetStats()
	start = time.Now()
	for it := tree.Iter(); it.Next(); {
	}
	stats = tree.Stats()
	fmt.Printf("  Iter %d keys:\t\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)

	// safe iterator
	tree.ResetStats()
	start = time.Now()
	for it := tree.IterSafe(); it.Next(); {
	}
	stats = tree.Stats()
	fmt.Printf("  ItertSafe %d keys:\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)

	// delete
	tree.ResetStats()
	start = time.Now()
	for _, k := range keys {
		tree.Delete(k)
	}
	stats = tree.Stats()
	fmt.Printf("  Delete %d keys:\t\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)
	assert.Equal(0, tree.Len())
	assertTreeCheck(t, tree, false)
}
