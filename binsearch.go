// The MIT License (MIT)
//
// Copyright (c) 2018 Fabian Wenzelmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package goel

import "sort"

type uintSlice []uint

func (p uintSlice) Len() int {
	return len(p)
}

func (p uintSlice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p uintSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func InsertedSorted(values []uint, newValues map[uint]struct{}) []uint {
	newSlice := make([]uint, len(newValues))
	i := 0
	for value, _ := range newValues {
		newSlice[i] = value
		i++
	}
	sort.Sort(uintSlice(newSlice))
	return unionSorted(values, newSlice)
}

func unionSorted(a, b []uint) []uint {
	n, m := len(a), len(b)
	res := make([]uint, 0, n+m)
	i, j := 0, 0
L:
	for {
		switch {
		case i < n && j < m:
			switch {
			case a[i] < b[j]:
				res = append(res, a[i])
				i++
			case a[i] > b[j]:
				res = append(res, b[j])
				j++
			default:
				res = append(res, a[i])
				i++
				j++
			}
		case i < n:
			res = append(res, a[i])
			i++
		case j < m:
			res = append(res, b[j])
			j++
		default:
			break L
		}
	}
	return res
}

func ContainsSorted(a []uint, x uint) bool {
	n := len(a)
	return sort.Search(n, func(i int) bool {
		return a[i] >= x
	}) == n
}
