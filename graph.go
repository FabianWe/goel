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

type ExtendedReachabilitySearch func(g ConceptGraph, goals map[uint]struct{}, start ...uint) map[uint]struct{}

func BFSToSet(g ConceptGraph, goals map[uint]struct{}, start ...uint) map[uint]struct{} {
	visited := make(map[uint]struct{}, len(start))
	result := make(map[uint]struct{}, len(goals))
	// very much the same as BFS, but we don't stop once a goal has been found
	// but continue until all reachable states from goal are found
	for _, value := range start {
		visited[value] = struct{}{}
	}
	// TODO again, what is the best place to copy?
	queue := start
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		// if next is a goal we can simply add the node
		if _, isGoal := goals[next]; isGoal {
			result[next] = struct{}{}
		}
		// first check if we can already stop (this is the case if we found all
		// goals)
		if len(result) == len(goals) {
			return result
		}
		// expand node
		ch := make(chan uint, 1)
		go g.Succ(next, ch)
		for v := range ch {
			if _, in := visited[v]; !in {
				visited[v] = struct{}{}
				queue = append(queue, v)
			}
		}
	}
	return result
}

type ExtendedGraphSearcher struct {
	extendedSearch ExtendedReachabilitySearch
	search         ReachabilitySearch
	start          []uint
}

func NewExtendedGraphSearcher(extendedSearch ExtendedReachabilitySearch,
	search ReachabilitySearch, bc *ELBaseComponents) *ExtendedGraphSearcher {
	start := make([]uint, bc.Nominals+1)
	var i uint
	for ; i < bc.Nominals; i++ {
		nominal := NewNominalConcept(i)
		start[i+1] = nominal.NormalizedID(bc)
	}
	return &ExtendedGraphSearcher{extendedSearch, search, start}
}

func NewExtendedBFSSearcher(bc *ELBaseComponents) *ExtendedGraphSearcher {
	return NewExtendedGraphSearcher(BFSToSet, BFS, bc)
}

func (searcher *ExtendedGraphSearcher) Search(g ConceptGraph, additionalStart, goal uint) bool {
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start[0] = additionalStart
	return searcher.search(g, goal, start...)
}

func (searcher *ExtendedGraphSearcher) ExtendedSearch(g ConceptGraph,
	additionalStart uint, goals map[uint]struct{}) map[uint]struct{} {
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start[0] = additionalStart
	return searcher.extendedSearch(g, goals, start...)
}

type ExtendedBidrectionalSearch struct {
	SearchType BidirectionalSearch
	Node       uint
}

func NewExtendedBidrectionalSearch(searchT BidirectionalSearch, node uint) *ExtendedBidrectionalSearch {
	return &ExtendedBidrectionalSearch{
		SearchType: searchT,
		Node:       node,
	}
}

func (searcher *ExtendedGraphSearcher) BidirectionalSearch(g ConceptGraph, c uint,
	goals map[uint]struct{}) []*ExtendedBidrectionalSearch {
	// TODO maybe add a parameter somewhere to control the number of go routines
	res := make([]*ExtendedBidrectionalSearch, 0, len(goals))

	// this part is for the direct (extended) search, we report back to done once
	// this is finished and store the result in directRes
	var directRes map[uint]struct{}
	done := make(chan bool)
	go func() {
		directRes = searcher.ExtendedSearch(g, c, goals)
		done <- true
	}()

	// now we start a go routine for all goals (and make a search to C)
	type searchResult struct {
		start uint
		found bool
	}

	ch := make(chan searchResult)
	for start, _ := range goals {
		go func(s uint) {
			found := searcher.Search(g, s, c)
			ch <- searchResult{s, found}
		}(start)
	}

	for i := 0; i < len(goals); i++ {
		next := <-ch
		if next.found {
			res = append(res, NewExtendedBidrectionalSearch(BidrectionalReverse, next.start))
		}
	}

	// now we have everything stored in res, wait for the direct search to finish
	// as well
	<-done
	// now iterate over result again, check the node was found both ways, that is
	// C ↝ D and D ↝ C, in this case update the value
	// all other values we will be added later on
	for _, bi := range res {
		// now check if value is in directRes as well
		if _, has := directRes[bi.Node]; has {
			bi.SearchType = BidrectionalBoth
			// remove from set so it won't get added later
			delete(directRes, bi.Node)
		}
	}
	// now everything that is left in directRes must be added as well, all those
	// values are not present int res yet
	for node, _ := range directRes {
		res = append(res, NewExtendedBidrectionalSearch(BidrectionalDirect, node))
	}
	return res
}
