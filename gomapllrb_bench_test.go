//go:build bench

package gomapllrb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBenchmarkRandom(t *testing.T) {
	title("Test perfmance / random")
	num := 1000000
	keys := make([]uint32, num, num)
	for i := 0; i < num; i++ {
		keys[i] = hash32(i)
	}
	perfTest(t, keys)
}

func TestBenchmarkAscending(t *testing.T) {
	title("Test perfmance / ascending")
	num := 1000000
	keys := make([]uint32, num, num)
	for i := 0; i < num; i++ {
		keys[i] = uint32(i)
	}
	perfTest(t, keys)
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
	fmt.Printf("  Put %d keys:\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)
	assert.Less(0, tree.Len())
	assertTreeCheck(t, tree, false)

	// find
	tree.ResetStats()
	start = time.Now()
	for _, k := range keys {
		tree.Exist(k)
	}
	stats = tree.Stats()
	fmt.Printf("  Find %d keys:\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)

	// iterator
	tree.ResetStats()
	start = time.Now()
	for it := tree.Iter(); it.Next(); {
		// let it loop
	}
	stats = tree.Stats()
	fmt.Printf("  Iter %d keys:\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)

	// delete
	tree.ResetStats()
	start = time.Now()
	for _, k := range keys {
		tree.Delete(k)
	}
	stats = tree.Stats()
	fmt.Printf("  Delete %d keys:\t%vms (%v)\n", len(keys), time.Since(start).Milliseconds(), stats)
	assert.Equal(0, tree.Len())
	assertTreeCheck(t, tree, false)
}
