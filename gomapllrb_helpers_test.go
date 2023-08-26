package gomapllrb

import (
	"encoding/binary"
	"fmt"
	"log"
	"testing"

	"github.com/spaolacci/murmur3"
	"golang.org/x/exp/constraints"
)

const (
	VERBOSE = false // enable visual inspections
)

func title(str string) {
	fmt.Printf("* TEST : %s\n", str)
}

func hash32(num int) uint32 {
	numBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(numBytes, uint32(num))
	hash := murmur3.New32()
	hash.Write(numBytes)
	return hash.Sum32()
}

func assertTreeCheck[K constraints.Ordered](t interface{}, tree *Tree[K], verbose bool) {
	if err := tree.Check(); err != nil {
		switch t.(type) {
		case *testing.T:
			t.(*testing.T).Error(err)
		case *testing.B:
			t.(*testing.B).Error(err)
		default:
			panic(err)
		}
		log.Println("ERROR:", err)
		verbose = true
	}
	if verbose {
		fmt.Print(tree)
		fmt.Printf("(#nodes %d, #red %d, #black %d\n", tree.Len(), 0, 0)
	}
}
