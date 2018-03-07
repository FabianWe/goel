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

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
)

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

func (g *SetGraph) String() string {
	strs := make([]string, len(g.graph))
	for i, s := range g.graph {
		strs[i] = fmt.Sprintf("%d ↦ %s", i, StringUintSet(s))
	}
	return fmt.Sprintf("{ %s }", strings.Join(strs, ",\n"))
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

// TODO there seems to be an infinite loop somewhere...
// TODO could easily be turned into a more concurrent version
// TODO think about the extended approach again...
// TODO maybe stop a node from being expanded if it has been expanded before?
// that is also true in the extended search.
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
				visited[v] = false
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

func prepareSearchStart(start []uint, additionalStart uint) []uint {
	// do duplicate check, that is check if additionalStart is already present
	// in start
	// since start is started we can simply compare it to the first and last
	// value: if it is in that range we don't need to add the additional start.
	// first do some boundary checking, if there are no nominals we might not
	// be able to access the first element in the array.
	if len(start) == 1 {
		// just the additional value is present in start, so that's okay,
		// no duplicate found
		start[0] = additionalStart
	} else {
		// now there is at least nominal, so now check the range
		fstNominal, lastNominal := start[1], start[len(start)-1]
		if additionalStart >= fstNominal && additionalStart <= lastNominal {
			// duplicate detected
			// don't add this element, simply remove first element from start slice
			start = start[1:]
		} else {
			// no duplicate
			start[0] = additionalStart
		}
	}
	return start
}

func (searcher *GraphSearcher) Search(g ConceptGraph, additionalStart, goal uint) bool {
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start = prepareSearchStart(start, additionalStart)
	// TODO remove prints
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
	var first, second bool
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		first = searcher.Search(g, c, d)
		wg.Done()
	}()
	go func() {
		second = searcher.Search(g, d, c)
		wg.Done()
	}()
	wg.Wait()
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

// Extended Search
type ExtendedReachabilitySearch func(g ConceptGraph, goals map[uint]struct{}, start ...uint) map[uint]struct{}

// TODO think about this again, seems rather slow... but why? hmmm...
func BFSToSet(g ConceptGraph, goals map[uint]struct{}, start ...uint) map[uint]struct{} {
	// same trick as in BFS
	result := make(map[uint]struct{}, len(goals))
	if len(goals) == 0 {
		return result
	}
	visited := make(map[uint]bool, len(start))
	// very much the same as BFS, but we don't stop once a goal has been found
	// but continue until all reachable states from goals are found
	for _, value := range start {
		visited[value] = true
	}
	// TODO again, what is the best place to copy?
	queue := start
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		// if next is a goal we add the goal, but again the same precautions as in
		// BFS.
		if _, isGoal := goals[next]; isGoal {
			visitedVal, wasVisited := visited[next]
			if wasVisited {
				if !visitedVal {
					// add element
					result[next] = struct{}{}
				}
			} else {
				// add
				result[next] = struct{}{}
			}
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
			if wasStart, wasVisited := visited[v]; !wasVisited || wasStart {
				visited[v] = false
				queue = append(queue, v)
			}
		}
	}
	return result
}

type ExtendedGraphSearcher struct {
	extendedSearch ExtendedReachabilitySearch
	start          []uint
}

func NewExtendedGraphSearcher(extendedSearch ExtendedReachabilitySearch,
	bc *ELBaseComponents) *ExtendedGraphSearcher {
	start := make([]uint, bc.Nominals+1)
	var i uint
	for ; i < bc.Nominals; i++ {
		nominal := NewNominalConcept(i)
		start[i+1] = nominal.NormalizedID(bc)
	}
	return &ExtendedGraphSearcher{extendedSearch, start}
}

func (searcher *ExtendedGraphSearcher) Search(g ConceptGraph, goals map[uint]struct{}, additionalStart uint) map[uint]struct{} {
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start = prepareSearchStart(start, additionalStart)
	return searcher.extendedSearch(g, goals, start...)
}

func (searcher *ExtendedGraphSearcher) BidrectionalSearch(g ConceptGraph, oldElements map[uint]struct{}, newElement uint) map[uint]BidirectionalSearch {
	res := make(map[uint]BidirectionalSearch, len(oldElements))
	// first we do an extended search from [{a} for each {a}] (that is what is
	// stored in start) to {k1, ..., kn, k}
	// This way we can easily determine if k was already reached (no searches
	// from ki -> k required, simply add them) and for which ki we already have
	// k -> ki (only to all others a search is required)
	// we use a channel to synchronize all the update of entries in the result
	// dict and a wait group to wait until all operations are done
	// once all elements have been added to the channel we close the done channel
	// but first create a copy of the old elements and add k to it
	firstGoals := make(map[uint]struct{}, len(oldElements)+1)
	firstGoals[newElement] = struct{}{}
	// copy old entries
	for oldValue, _ := range oldElements {
		firstGoals[oldValue] = struct{}{}
	}
	// now run the first search, that is simply from the old start
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start = start[1:]
	alreadyReached := searcher.extendedSearch(g, firstGoals, start...)
	// now initialize wait group and channel and start a function that waits
	// on updates on that channel
	type searchRes struct {
		value uint
		res   BidirectionalSearch
	}

	ch := make(chan searchRes, 1)
	done := make(chan bool)
	// start a listener, we define an internal update function as well

	// this function assumes that BidirectionalFalse is never added to res
	updateEntry := func(value uint, sRes BidirectionalSearch) {
		if oldEntry, hasEntry := res[value]; hasEntry {
			if oldEntry != sRes {
				res[value] = BidrectionalBoth
			}
		} else {
			res[value] = sRes
		}
	}

	go func() {
		for entry := range ch {
			updateEntry(entry.value, entry.res)
		}
		done <- true
	}()

	// this set stores all ki for which we cannot yet determine if k -> ki
	searchRequired := make(map[uint]struct{})

	// for each ki we add 1 to the wait group, thus len(oldElements) is added
	var wg sync.WaitGroup
	wg.Add(len(oldElements))

	_, kReached := alreadyReached[newElement]
	// a goal set that contains only k
	kGoalSet := map[uint]struct{}{newElement: struct{}{}}

	for ki, _ := range oldElements {
		// check k -> ki
		if _, kiReached := alreadyReached[ki]; kiReached {
			// if it is already reached just add it
			ch <- searchRes{ki, BidrectionalDirect}
		} else {
			// a search to ki is required
			searchRequired[ki] = struct{}{}
		}
		go func(ki uint) {
			defer wg.Done()
			// now check for ki -> k, this may require a search and therefor runs
			// in a goroutine s.t. may searches can run concurrently
			// we use wg to sync.
			if kReached {
				// just add it, no search required
				ch <- searchRes{ki, BidrectionalReverse}
			} else {
				// perform a search [ki] -> {k}
				kiRes := searcher.extendedSearch(g, kGoalSet, ki)
				// check if k was found
				if len(kiRes) > 0 {
					// TODO once tested
					if len(kiRes) != 1 {
						log.Println("Weird search result!")
					}
					// search was a success, add element
					ch <- searchRes{ki, BidrectionalReverse}
				}
			}
		}(ki)
	}
	// now wait for all searches that might be running
	wg.Wait()
	// close the channel
	close(ch)
	// now wait until all adds have happened
	<-done
	// no more searches required
	if len(searchRequired) == 0 {
		return res
	}
	// now searchRequired contains all ki to witch a search must be performed
	// check which elements are reachable now
	reachableNow := searcher.extendedSearch(g, searchRequired, newElement)
	for ki, _ := range reachableNow {
		// just update entry
		updateEntry(ki, BidrectionalDirect)
	}
	// now we're done
	return res
}

func (searcher *ExtendedGraphSearcher) FindConnectedPairs(g ConceptGraph, s map[uint]struct{}) *BCPairSet {
	res := NewBCPairSet(nil, uint(len(s)))
	// first we do an extended search from [{a} for each {a}] (that is what is
	// stored in start) to s
	// this way we can easily determine which D all already reached from each C
	// now run the first search, that is simply from the old start
	start := make([]uint, len(searcher.start))
	copy(start, searcher.start)
	start = start[1:]
	alreadyReached := searcher.extendedSearch(g, s, start...)

	// this set contains all elements that were not reached by the initial
	// search and still must be searched
	// max is not really required here, but just to be sure.
	searchRequired := make(map[uint]struct{}, IntMax(0, len(s)-len(alreadyReached)))
	// now fill searchRequired with all elements from goal not yet found
	for d, _ := range s {
		if _, reached := alreadyReached[d]; !reached {
			searchRequired[d] = struct{}{}
		}
	}
	// now for each c perform a search to search required and unite the goals
	// we use a wait group and a listener function to synchronize everything
	type searchRes struct {
		c               uint
		additionalGoals map[uint]struct{}
	}
	ch := make(chan searchRes, 1)
	done := make(chan bool)
	// start a listener that adds entries to the result
	go func() {
		for update := range ch {
			c, additionalGoals := update.c, update.additionalGoals
			for d, _ := range additionalGoals {
				res.AddID(c, d)
			}
		}
		done <- true
	}()

	// create a wait group, this group waits until all searches from all C ->
	// searchRequired are done
	var wg sync.WaitGroup
	wg.Add(len(s))
	for c, _ := range s {
		// now search from c to searchRequired
		go func(c uint) {
			defer wg.Done()
			additionalGoals := searcher.extendedSearch(g, searchRequired, c)
			ch <- searchRes{c, additionalGoals}
			// also add values that are already found
			ch <- searchRes{c, alreadyReached}
		}(c)
	}
	// wait for all searches that might be running
	wg.Wait()
	// close channel
	close(ch)
	// wait until all adds have happened
	<-done
	return res
}

// Transitive closure graph.

type TransitiveClosureGraph struct {
	*SetGraph
	closure *Relation
}

func NewTransitiveClosureGraph() *TransitiveClosureGraph {
	return &TransitiveClosureGraph{
		SetGraph: NewSetGraph(),
		closure:  NewRelation(0),
	}
}

func (g *TransitiveClosureGraph) Init(numConcepts uint) {
	g.SetGraph.Init(numConcepts)
	g.closure = NewRelation(numConcepts)
}

func (g *TransitiveClosureGraph) AddEdge(source, target uint) bool {
	if !g.SetGraph.AddEdge(source, target) {
		return false
	}
	// now we have a new edge from source to target
	// that means that all nodes reaching source also reach everything
	// that target reaches including source as well.
	// that is: they all need an update
	// we actually store all targets in a set, this way we don't iterate over
	// closure while adding to it... this might break things
	// we can compute both slices concurrently
	newTargets := map[uint]struct{}{target: struct{}{}}
	updateNodes := map[uint]struct{}{source: struct{}{}}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// here we update all new target nodes, that is everything that is reachable
		// from target
		for reachable, _ := range g.closure.Mapping[target] {
			newTargets[reachable] = struct{}{}
		}
	}()
	go func() {
		defer wg.Done()
		// here we compute all nodes that have a path to source
		for reachesSource, _ := range g.closure.ReverseMapping[source] {
			updateNodes[reachesSource] = struct{}{}
		}
	}()
	wg.Wait()
	// now that we have the mappings we have to perform the update...
	for v1, _ := range updateNodes {
		// insert all targets
		for v2, _ := range newTargets {
			g.closure.Add(v1, v2)
		}
	}
	return true
}

func ClosureToSet(g ConceptGraph, goals map[uint]struct{}, start ...uint) map[uint]struct{} {
	// TODO ugly as hell... but well...
	tcg, ok := g.(*TransitiveClosureGraph)
	if !ok {
		log.Println("ClosureToSet called for something that is not a TransitiveClosureGraph! Type:", reflect.TypeOf(g))
		return BFSToSet(g, goals)
	}
	res := make(map[uint]struct{}, len(goals))
	// just build the union of everything reachable by some start
	// TODO: speedup: If len(start) == 1 no copying is required, right?
	// The other methods never change the returned map, or do they?
	allReachable := make(map[uint]struct{}, len(goals))
	for _, v := range start {
		// now add everything that v reaches to allReachable
		for reachable, _ := range tcg.closure.Mapping[v] {
			allReachable[reachable] = struct{}{}
		}
	}
	// now iterate over all goals we're looking for and add those reachable
	// from anything in start
	for goal, _ := range goals {
		if _, reachable := allReachable[goal]; reachable {
			res[goal] = struct{}{}
		}
	}
	return res
}
