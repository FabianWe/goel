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

// TODO: It's important that the search can easily assume that start is
// unique, i.e. no duplicates.
// That is required for the reachability search that requires path lengths
// >= 1.
type ReachabilitySearch func(g ConceptGraph, goal uint, start ...uint) bool

// TODO could easily be turned into a more concurrent version
// TODO think about the extended approach again...
func BFS(g ConceptGraph, goal uint, start ...uint) bool {
	// visited stores entries which have already been visited
	// but as an addition to the "normal" BFS we don't simply store which
	// elements have been visited but also whether they were added here because
	// they're a start node or because the node appeared during an expansion.
	//
	// That is: was the node added during the start (in which case we will
	// map the node to true) or in some later run (then we map to false).
	// This way we have the following: If there is an entry for a node we can
	// easily check if it was "visited" because it's a start node or if it was
	// visited because it was added during some expansion.
	visited := make(map[uint]bool, len(start))
	// map each start node to true
	for _, value := range start {
		visited[value] = true
	}
	queue := start
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if next == goal {
			// the standard BFS would stop here, but we want to know if this node
			// really was added during and an expansion before
			visitedVal, wasVisited := visited[next]
			if wasVisited {
				// node was visited before, now if this node was visited because it
				// encountered during an expansion we're done (during an expansion
				// meaning that the value is false).
				if !visitedVal {
					return true
				}
				// no else case here required, node was reached but just because it
				// was in visited from the start
				// we have to expand the node.
			} else {
				// if the node was not present in the visited map we can safely assume
				// that it was not one of the start nodes
				return true
			}
		}
		// we haven't reached the goal yet, so expand the node
		ch := make(chan uint, 1)
		go g.Succ(next, ch)
		for v := range ch {
			// Now things become a bit more complicated than in the normal version.
			// We want to add a node to the queue even if it was already visited
			// before but this was only because the node was a start node.
			// In this case we simply add it again.
			// So we want to add v to the queue again if:
			// (1) It was not visited before
			// (2) It was visited before but only because it was a start node
			if wasStart, wasVisited := visited[v]; !wasVisited || wasStart {
				// add node and mark as visited during an expansion
				visited[v] = true
				queue = append(queue, v)
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

// TODO add duplicate check, i.e. check if additionalStart is a nominal, this
// should only require some lookups in the start slice
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
