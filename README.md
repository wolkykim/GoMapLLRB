GoMapLLRB [Left-leaning red-black tree in Go]
=============================================

[![CI Status](https://github.com/wolkykim/gomapllrb/workflows/CI/badge.svg)](https://github.com/wolkykim/gomapllrb/actions) [![Code Coverage](https://github.com/wolkykim/gomapllrb/wiki/coverage.svg)](https://raw.githack.com/wiki/wolkykim/gomapllrb/coverage.html) [![Go Report Card](https://goreportcard.com/badge/github.com/wolkykim/gomapllrb)](https://goreportcard.com/report/github.com/wolkykim/gomapllrb) [![API Reference](https://pkg.go.dev/badge/github.com/wolkykim/gomapllrb.svg)](https://pkg.go.dev/github.com/wolkykim/gomapllrb)

GoMapLLRB is a Go package that implements an in-memory key/value store using LLRB algorithm.
LLRB(Left-Leaning Red-Black) is a self-balancing binary search tree that keeps the keys in order,
allowing ordered iteration and finding the nearest keys.

This is a GoLang version of the C implementation in
[qLibc](https://github.com/wolkykim/qlibc) library.

# GoMapLLRB vs Built-in map

|                     | GoMapLLRB    | Built-in map |
| ------------------- | :----------: | :----------: |
| Data structure      | Binary Tree  | Hash Table   |
| Iteration Order     | Ordered      | Random       |
| Find nearest key    | Yes          | No           |
| Performance         | O(log n)     | O(1)         |
| Memory overhead     | O(n)         | O(n)         |

# Usages [![API Reference](https://pkg.go.dev/badge/github.com/wolkykim/gomapllrb.svg)](https://pkg.go.dev/github.com/wolkykim/gomapllrb)

### Simple Example

```go
import "github.com/wolkykim/gomapllrb"

t := gomapllrb.New[string]()
t.Put("foo", "Hello World")
fmt.Println(t.Get("foo"))
t.Delete("foo")

[Output]
Hello World
```
[[Play the code](https://go.dev/play/p/lu33sWg1zdz)]

Other getter methods: Exist(), Min(), Max(), Bigger(), Smaller(), EqualOrBigger(), EqualOrSmaller(), ...
See [API documents](https://pkg.go.dev/github.com/wolkykim/gomapllrb#section-documentation) for details.

### Iteration

```go
t := New[int]()
for _, k := range []int{7, 1, 3, 9, 5} {
    t.Put(k, k*10)
}

for it := t.Iter(); it.Next(); {
    fmt.Printf("%d=%d ", it.Key(), it.Val())
}

for it := t.Range(3, 8); it.Next(); {
    fmt.Printf("%d:%d ", it.Key(), it.Val())
}

[Output]
1=10 3=30 5=50 7=70 9=90 3:30 5:50 7:70
```
[[Play the code](https://go.dev/play/p/ua19jQ6WnaS)]

### Students on DSA course
```go
fmt.Println(t, t.Stats())

[Output]
    ┌──[9]
┌───7
│   └──[5]
3
└───1
 Variant:LLRB234, Put:5, Delete:0, Get:0, Rotate:0.80, Flip:0.20
```
[[Play the code](https://go.dev/play/p/TssSkvYvmV-)]

# Performance 2-3-4 LLRB Vs. 2-3 LLRB

For anyone curious, here's the performance test result between 2-3-4 LLRB and 2-3 LLRB.
Tested on 2021 Apple M1 Pro 10-core MacBook. GoMapLLRB supports 2-3-4 and 2-3 LLRB and
ships by default to balance the tree structure in the 2-3-4 variant.

|                | 2-3-4 LLRB | 2-3 LLRB   | | 2-3-4 LLRB | 2-3 LLRB   | | 2-3-4 LLRB | 2-3 LLRB   |
| ---------------| ---------: | ---------: |-| ---------: | ---------: |-| ---------: | ---------: |
| Workload       | 1 million  | 1 million  | | 3 million  | 3 million  | | 10 million | 10 million |
|                |            |            | |            |            | |            |            |
| Insert         |      602ms |      654ms | |     2838ms |     3185ms | |    13065ms |    13412ms |
| Lookup         |      376ms |      391ms | |     1860ms |     1869ms | |     9480ms |    10647ms |
| Delete         |      693ms |      702ms | |     3201ms |     3166ms | |    14199ms |    15594ms |
|                |            |            | |            |            | |            |            |
| Rotations(Ins) |       1.08 |       1.19 | |       1.08 |       1.19 | |       1.08 |       1.19 |
| Rotations(Del) |      16.46 |      19.45 | |      17.43 |      21.77 | |      18.64 |      23.53 |
| Flips(Ins)     |       0.57 |       0.75 | |       0.57 |       0.75 | |       0.57 |       0.75 |
| Flips(Del)     |       3.31 |      15.33 | |       3.44 |      17.42 | |       3.58 |      18.45 |

# Copyright

GoMapLLRG is published under 2-clause BSD license known as Simplified BSD License.
Please refer the LICENSE document included in the package for more details.

Wanna thank me? Give it a star to this project if it helps. That'll do it!
https://github.com/wolkykim/GoMapLLRB
