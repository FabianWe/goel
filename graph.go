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

package goel

type ConceptGraph interface {
	Init(numConcepts uint)
	AddEdge(source, target uint) bool
	Succ(vertex uint, ch chan<- uint)
}

type SetGraph struct {
	graph []map[uint]struct{}
}

func NewSetGraph() *SetGraph {
	return &SetGraph{graph: nil}
}

func (g *SetGraph) Init(numConcepts uint) {
	g.graph = make([]map[uint]struct{}, numConcepts)
	var i uint = 0
	for ; i < numConcepts; i++ {
		g.graph[i] = make(map[uint]struct{})
	}
}

func (g *SetGraph) AddEdge(source, target uint) bool {
	m := g.graph[source]
	oldLen := len(m)
	m[target] = struct{}{}
	return oldLen != len(m)
}

func (g *SetGraph) Succ(vertex uint, ch chan<- uint) {
	for succ, _ := range g.graph[vertex] {
		ch <- succ
	}
	close(ch)
}

type SliceGraph struct {
	Graph [][]uint
}

func NewSliceGraph() *SliceGraph {
	return &SliceGraph{nil}
}

func (g *SliceGraph) Init(numConcepts uint) {
	g.Graph = make([][]uint, numConcepts)
}

func (g *SliceGraph) AddEdge(source, target uint) bool {
	// not nice, but that's the drawback...
	for _, v := range g.Graph[source] {
		if v == target {
			return false
		}
	}
	// now add
	g.Graph[source] = append(g.Graph[source], target)
	return true
}

func (g *SliceGraph) Succ(vertex uint, ch chan<- uint) {
	for _, val := range g.Graph[vertex] {
		ch <- val
	}
	close(ch)
}

type ReachabilitySearch func(g ConceptGraph, goal uint, start ...uint) bool

// TODO could easily be turned into a more concurrent version
// TODO I think this is wrong. Because if we have an isolated node {a}
// this method will return true for some other node C in the graph.
// The problem is that everything in start ins considered reachable.
// Or is this even required by the algorithm? Read the proof again!
func BFS(g ConceptGraph, goal uint, start ...uint) bool {
	visited := make(map[uint]struct{}, len(start))
	for _, value := range start {
		visited[value] = struct{}{}
	}
	// queue := make([]uint, len(start))
	// copy(queue, start)
	// TODO where is the best place to create copy? document this as well!
	// is a copy required or is it safe not to use a copy?
	queue := start
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if next == goal {
			return true
		}
		// expand the node
		ch := make(chan uint, 1)
		go g.Succ(next, ch)
		for v := range ch {
			if _, in := visited[v]; !in {
				visited[v] = struct{}{}
				queue = append(queue, v)
				// note that we don't check here if we reached the goal
				// this is also important because otherwise ch might stay open
				// because succ closes the channel!
				// of course we could check if we added goal to the queue and
				// after all iterations are done return already true though
			}
		}
	}
	return false
}

type GraphSearcher struct {
	search ReachabilitySearch
	start  []uint
}

func NewGraphSearcher(search ReachabilitySearch, bc *ELBaseComponents) *GraphSearcher {
	start := make([]uint, bc.Nominals+1)
	var i uint
	for ; i < bc.Nominals; i++ {
		nominal := NewNominalConcept(i)
		start[i+1] = nominal.NormalizedID(bc)
	}
	return &GraphSearcher{search, start}
}

func (searcher *GraphSearcher) Search(g ConceptGraph, additionalStart, goal uint) bool {
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start[0] = additionalStart
	return searcher.search(g, goal, start...)
}

//go:generate stringer -type=BidirectionalSearch

type BidirectionalSearch int

const (
	BidirectionalFalse BidirectionalSearch = iota
	BidrectionalDirect
	BidrectionalReverse
	BidrectionalBoth
)

// TODO is this even required? See comment in NaiveSolver rule 6
func (searcher *GraphSearcher) BidrectionalSearch(g ConceptGraph, c, d uint) BidirectionalSearch {
	// run two searches concurrently
	ch := make(chan bool)
	go func() {
		ch <- searcher.Search(g, c, d)
	}()
	go func() {
		ch <- searcher.Search(g, d, c)
	}()
	first := <-ch
	second := <-ch
	switch {
	case first && second:
		return BidrectionalBoth
	case first:
		return BidrectionalDirect
	case second:
		return BidrectionalReverse
	default:
		return BidirectionalFalse
	}
}
