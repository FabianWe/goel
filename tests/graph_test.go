// The MIT License (MIT)
//
// Copyright (c) 2016, 2017 Fabian Wenzelmann
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
import "gkigit.informatik.uni-freiburg.de/fwenzelmann/goel"

var g1 *goel.SliceGraph = &goel.SliceGraph{[][]uint{[]uint{1, 2}, []uint{4}, []uint{3}, nil, nil}}
var g2 *goel.SliceGraph = &goel.SliceGraph{[][]uint{
	[]uint{1},
	[]uint{2},
	[]uint{3, 4, 5},
	[]uint{2, 6},
	[]uint{0},
	nil,
	[]uint{1, 7},
	[]uint{7},
},
}

func TestConnectedG1(t *testing.T) {
	tests := []struct {
		goal     uint
		start    []uint
		expected bool
	}{
		{3, []uint{0}, true},
		{3, []uint{1}, false},
		{4, []uint{4}, true},
	}
	for _, test := range tests {
		actual := goel.BFS(g1, test.goal, test.start...)
		if actual != test.expected {
			t.Errorf("Error while searching g1: Expected %v but got %v (search from %v to %d)",
				test.expected, actual, test.start, test.goal)
		}
	}
}

func TestConnectedG2(t *testing.T) {
	tests := []struct {
		goal     uint
		start    []uint
		expected bool
	}{
		{7, []uint{0}, true},
		{7, []uint{4}, true},
		{4, []uint{6}, true},
		{0, []uint{7}, false},
	}
	for _, test := range tests {
		actual := goel.BFS(g2, test.goal, test.start...)
		if actual != test.expected {
			t.Errorf("Error while searching g2: Expected %v but got %v (search from %v to %d)",
				test.expected, actual, test.start, test.goal)
		}
	}
}
