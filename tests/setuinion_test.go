// The MIT License (MIT)
//
// Copyright (c) 2016, 2017, 2018 Fabian Wenzelmann
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

package tests

import "testing"
import "github.com/FabianWe/goel"

// variables for testing

var aMap map[uint]struct{}
var bMap map[uint]struct{}

var aSlice []uint

// global result variables to prevent compiler optimisation
var mapRes map[uint]struct{}

var sliceRes []uint

var contains bool

func init() {
	aMap = make(map[uint]struct{}, 100000)
	bMap = make(map[uint]struct{}, 100000)

	aSlice = make([]uint, 100000)

	var i uint
	for ; i < 50000; i++ {
		aMap[i] = struct{}{}
		aSlice[i] = i
		bMap[i] = struct{}{}
	}
	i = 50000
	for ; i < 100000; i++ {
		val := 200000 + i
		aMap[val] = struct{}{}
		aSlice[i] = val
		bMap[i] = struct{}{}
	}
}

func mapUnion(a, b map[uint]struct{}) map[uint]struct{} {
	res := make(map[uint]struct{}, len(a)+len(b))
	for v, _ := range a {
		res[v] = struct{}{}
	}
	for v, _ := range b {
		res[v] = struct{}{}
	}
	return res
}

func mapContains(a map[uint]struct{}, x uint) bool {
	_, has := a[x]
	return has
}

func BenchmarkMapUnion(b *testing.B) {
	// prevent compiler optimisation
	var r map[uint]struct{}
	for n := 0; n < b.N; n++ {
		r = mapUnion(aMap, bMap)
	}
	mapRes = r
}

func BenchmarkSortedSliceUnion(b *testing.B) {
	// prevent compiler optimisation
	var r []uint
	for n := 0; n < b.N; n++ {
		r = goel.InsertedSorted(aSlice, bMap)
	}
	sliceRes = r
}

func BenchmarkContainsMap(b *testing.B) {
	var r bool
	for n := 0; n < b.N; n++ {
		r = mapContains(aMap, 40000)
	}
	contains = r
}

func BenchmarkNotContainsMap(b *testing.B) {
	var r bool
	for n := 0; n < b.N; n++ {
		r = mapContains(aMap, 600000)
	}
	contains = r
}

func BenchmarkSortedSliceContains(b *testing.B) {
	var r bool
	for n := 0; n < b.N; n++ {
		r = goel.ContainsSorted(aSlice, 40000)
	}
	contains = r
}

func BenchmarkSortedSliceNotContains(b *testing.B) {
	var r bool
	for n := 0; n < b.N; n++ {
		r = goel.ContainsSorted(aSlice, 600000)
	}
	contains = r
}
