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

package goel

// SuccFunction is a function that must produce all successors of the specified
// vertex in a directed graph.
// All successors must be written to the channel and the channel must be closed
// once all successors have been produced.
type SuccFunction func(vertex uint, ch chan<- uint)

// GraphSearcher is an interface for all algorithms that perform a graph search.
// This search must check if the goal state can be reached when starting in
// any of the specified start states.
// succ is used to generate all successors of a vertex.
type GraphSearcher interface {
	IsReachable(succ SuccFunction, goal uint, start ...uint) bool
}

type BFSearcher struct{}

func NewBFSearcher() BFSearcher {
	return BFSearcher{}
}

func (bfs BFSearcher) IsReachable(succ SuccFunction, goal uint, start ...uint) bool {
	visited := make(map[uint]struct{}, len(start))
	for _, value := range start {
		visited[value] = struct{}{}
	}
	queue := make([]uint, len(start))
	copy(queue, start)
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if next == goal {
			return true
		}
		// expand the node
		ch := make(chan uint, 1)
		go succ(next, ch)
		for v := range ch {
			if _, in := visited[v]; !in {
				visited[v] = struct{}{}
				queue = append(queue, v)
			}
		}
	}
	return false
}
