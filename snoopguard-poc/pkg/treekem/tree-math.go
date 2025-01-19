package treekem

import (
	"math"
	"slices"
)

func log2(x int) int {
	if x == 0 {
		return 0
	}
	return int(math.Log2(float64(x)))
}

func level(x int) int {
	if x&0x01 == 0 {
		return 0
	}

	k := 0
	for (x>>k)&0x01 == 1 {
		k++
	}
	return k
}

func nodeWidth(n int) int {
	return 2*(n-1) + 1
}

func assertInRange(x, n int) {
	if x > nodeWidth(n) {
		panic("Assertion assertInRange failed")
	}
}

func root(n int) int {
	w := nodeWidth(n)
	return 1<<log2(w) - 1
}

func left(x int) int {
	if level(x) == 0 {
		return x
	}
	return x ^ (0x01 << (level(x) - 1))
}

func right(x, n int) int {
	assertInRange(x, n)

	if level(x) == 0 {
		return x
	}

	r := x ^ (0x03 << (level(x) - 1))
	for r >= nodeWidth(n) {
		r = left(r)
	}
	return r
}

func parentStep(x int) int {
	k := level(x)
	return (x | (1 << k)) & ^(1 << (k + 1))
}

func parent(x, n int) int {
	assertInRange(x, n)

	if x == root(n) {
		return x
	}

	p := parentStep(x)
	for p >= nodeWidth(n) {
		p = parentStep(p)
	}
	return p
}

func sibling(x, n int) int {
	assertInRange(x, n)

	p := parent(x, n)
	if x < p {
		return right(p, n)
	} else if x > p {
		return left(p)
	}
	return p
}

func trDirpath(x, n int) []int {
	assertInRange(x, n)

	if x == root(n) {
		return []int{}
	}

	d := []int{x}
	p := parent(x, n)
	r := root(n)
	for p != r {
		d = append(d, p)
		p = parent(p, n)
	}
	return d
}

func trCopath(x, n int) []int {
	dirpath := trDirpath(x, n)
	copath := make([]int, len(dirpath))
	for i, x := range dirpath {
		copath[i] = sibling(x, n)
	}
	return copath
}

func frontier(n int) []int {
	if n <= 0 {
		panic("Assertion n > 0 failed")
	}

	last := 2 * (n - 1)
	f := trCopath(last, n)
	slices.Reverse(f)

	if len(f) == 0 || f[len(f)-1] != last {
		f = append(f, last)
	}

	for len(f) > 1 {
		r := f[len(f)-1]
		p := parent(r, n)
		if p != parentStep(r) {
			break
		}

		// Replace the last two nodes with their parent
		f = append(f[:len(f)-2], p)
	}

	return f
}

func shadow(x, n int) []int {
	h := level(x)
	L := x
	R := x
	for h > 0 {
		L = left(L)
		R = right(R, n)
		h--
	}

	shadow := make([]int, R-L+1)
	for i := 0; i < len(shadow); i++ {
		shadow[i] = L + i
	}
	return shadow
}

func leaves(n int) []int {
	leaves := make([]int, n)
	for i := 0; i < n; i++ {
		leaves[i] = 2 * i
	}
	return leaves
}
