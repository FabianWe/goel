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

type ConceptGraph interface {
	Init(numConcepts uint)
	AddEdge(source, target uint)
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

func (g *SetGraph) AddEdge(source, target uint) {
	g.graph[source][target] = struct{}{}
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

func (g *SliceGraph) AddEdge(source, target uint) {
	g.Graph[source] = append(g.Graph[source], target)
}

func (g *SliceGraph) Succ(vertex uint, ch chan<- uint) {
	for _, val := range g.Graph[vertex] {
		ch <- val
	}
	close(ch)
}

type ReachabilitySearch func(g ConceptGraph, goal uint, start ...uint) bool

type GraphSearcher struct {
	g        ConceptGraph
	search   ReachabilitySearch
	nominals []uint
}

func NewGraphSearcher(g ConceptGraph, search ReachabilitySearch, nominals []uint) *GraphSearcher {
	internalNominals := make([]uint, len(nominals)+1)
	copy(internalNominals[1:], nominals)
	return &GraphSearcher{
		g:        g,
		search:   search,
		nominals: internalNominals,
	}
}

func (searcher *GraphSearcher) Search(additionalStart, goal uint) bool {
	searcher.nominals[0] = additionalStart
	return searcher.search(searcher.g, goal, searcher.nominals...)
}

func BFS(g ConceptGraph, goal uint, start ...uint) bool {
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
