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

import "sync"

// SUpdate is a type that stores the information that D has been added to S(C).
// It is usually used in a queue that stores all updates that still must be
// executed (notifications for that update must be issued).
type SUpdate struct {
	C, D uint
}

// NewSUpdate creates a new SUpdate.
func NewSUpdate(c, d uint) *SUpdate {
	return &SUpdate{C: c, D: d}
}

// RUpdate is a type that stores the information that (C, D) has been added to
// r. Similar to SUpdate it is usually used in a queue that stores updates that
// still must be executed (notifcations for that update must be issued).
type RUpdate struct {
	R, C, D uint
}

// NewRUpdate returns a new RUpdate.
func NewRUpdate(r, c, d uint) *RUpdate {
	return &RUpdate{
		R: r,
		C: c,
		D: d,
	}
}

// AllChangesState extends the StateHandler interface as mentioned in the
// comment there.
// This is the version in which the graph only checks if an edge was added.
// It has additional methods for updating the graph (add an edge between C and
// D), check reachability of two concepts and test if S(C) ⊆ S(D).
//
// A default implementation is given in AllChangesSolverState.
// TODO add name of CR6 here.
type AllChangesState interface {
	StateHandler

	// SubsetConcepts(c, d uint) bool
	// UpdateGraph(c, d uint) bool
	// TODO add search method(s) here.
	ExtendedSearch(goals map[uint]struct{}, additionalStart uint) map[uint]struct{}
	// TODO describe requirements
	AddSubsetRule(c, d uint, ch <-chan bool)
}

type AllChangesSolverState struct {
	*SolverState

	Graph      ConceptGraph
	Searcher   *ExtendedGraphSearcher
	graphMutex *sync.RWMutex
}

func NewAllChangesSolverState(c *ELBaseComponents, g ConceptGraph, search ExtendedReachabilitySearch) *AllChangesSolverState {
	var graphMutex sync.RWMutex
	// initialize solver state, graph and searcher
	res := AllChangesSolverState{
		SolverState: nil,
		Graph:       g,
		Searcher:    nil,
		graphMutex:  &graphMutex,
	}
	// initalize SolverState, graph and searcher concurrently
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		res.SolverState = NewSolverState(c)
	}()
	go func() {
		defer wg.Done()
		// we use + 1 here because we want to use the normalized id directly, so
		// the bottom concept must be taken into consideration
		numBCD := c.NumBCD() + 1
		res.Graph.Init(numBCD)
	}()
	go func() {
		defer wg.Done()
		res.Searcher = NewExtendedGraphSearcher(search, c)
	}()

	wg.Wait()
	return &res
}

func (state *AllChangesSolverState) ExtendedSearch(goals map[uint]struct{},
	additionalStart uint) map[uint]struct{} {
	state.graphMutex.RLock()
	res := state.Searcher.Search(state.Graph, goals, additionalStart)
	state.graphMutex.RUnlock()
	return res
}

// func (state *AllChangesSolverState) UpdateGraph(c, d uint) bool {
// 	state.graphMutex.Lock()
// 	res := state.Graph.AddEdge(c, d)
// 	state.graphMutex.Unlock()
// 	return res
// }

// func (state *AllChangesSolverState) IsReachable(c, d uint) bool {
// 	state.graphMutex.RLock()
// 	res := state.Searcher.Search(state.Graph, c, d)
// 	state.graphMutex.RUnlock()
// 	return res
// }
//
// func (state *AllChangesSolverState) BidrectionalSearch(c, d uint) BidirectionalSearch {
// 	state.graphMutex.RLock()
// 	res := state.Searcher.BidrectionalSearch(state.Graph, c, d)
// 	state.graphMutex.RUnlock()
// 	return res
// }

type AllGraphChangeHandler interface {
	GetGraphNotification(state AllChangesState) bool
}

// AllChangesSNotification is a special handler type for rule CR6.
// We're interested in updates on all {a} for all S(C) and S(D). Thus
// storing this in the proposed pattern for SNotifications would require much
// memory.
// So we do the following: Wait until some set S(C) gets updated with C'.
// If C' is not of the form {a} do nothing.
// If it is of the form perform the test and apply the rule if required.
// Note that here we use the extendted AllChangesState interface, not
// just SolverState.
// So this interface is used to show the difference between SNotification
// and to use the extended state interface.
type AllChangesSNotification interface {
	// Information, that C' was added to S(C)
	GetSNotification(state AllChangesState, c, cPrime uint) bool
}

// TODO I'm so totally not sure if this is correct.
// We add a map here that maps for each {a} to a list of all C with {a} ∈ S(C).
// This requires a bit more memory but I think(!) that it's worth it. Otherwise
// we always have to iterate over all S(D) and test where {a} is contained.
// This way finding all C, D with {a} ∈ S(C) ⊓ S(D) is easy.
type AllChangesCR6 struct {
	// TODO use slice here, is much nicer, but well it also works this way...
	aMap map[uint]map[uint]struct{}
	// TODO is this required? Think about it...
	aMutex *sync.Mutex
}

func NewAllChangesCR6() *AllChangesCR6 {
	var m sync.Mutex
	return &AllChangesCR6{aMap: make(map[uint]map[uint]struct{}, 10), aMutex: &m}
}

func (n *AllChangesCR6) applyRule(state AllChangesState, goals map[uint]struct{}, c uint) bool {
	connected := state.ExtendedSearch(goals, c)
	result := false
	for d, _ := range connected {
		// no need to do anyhting if c == d
		if c == d {
			continue
		}
		// now we found a connection between C and D, that is now we have
		// C ↝ D
		// so now we can just union both concepts and add a new rule
		// TODO is size 1 okay? should be
		done := make(chan bool, 1)
		state.AddSubsetRule(c, d, done)
		result = state.UnionConcepts(c, d) || result
		done <- true
	}
	return result
}

func (n *AllChangesCR6) GetGraphNotification(state AllChangesState) bool {
	// if the graph has changed we iterate over all pairs and revaulate
	// the condition, that is we add new rules etc.
	// maybe there are nicer ways but we'll do the following:
	// iterate over each {a} and then perform the extended search for each C
	// that contains {a}.

	// lock mutex
	n.aMutex.Lock()
	defer n.aMutex.Unlock()

	result := false
	for _, containedIn := range n.aMap {
		for c, _ := range containedIn {
			result = n.applyRule(state, containedIn, c) || result
		}
	}
	return result
}

func (n *AllChangesCR6) GetSNotification(state AllChangesState, c, cPrime uint) bool {
	// first check if a nominal was added, otherwise just ignore the update
	concept := state.GetComponents().GetConcept(cPrime)
	// try to convert to nominal concept
	if _, ok := concept.(NominalConcept); !ok {
		// not interested in update
		return false
	}
	// now we're interested in the update
	// in order to do so we must iterate over all elements where {a} is contained
	// (this is all elements in the intersection)
	// and perform a reachability search.
	// to make it concurrency safe we completely lock the mutex
	// TODO is there a nicer way? This should work anyway...
	n.aMutex.Lock()
	defer n.aMutex.Unlock()
	// now we only have to perform a search from C to all D with {a} ∈ S(D):
	// This is the only new information we have, we don't have to worry about the
	// "old" elements in the set, a connection between them is not affected by
	// the information that {a} was added to S(C): We've already performed a
	// search for those elements, if the graph changes we will reconsider
	// so first get all D in which {a} is contained
	// we can use the extended search method for that, it will give us all pairs
	// that are connected when starting the search with C
	result := n.applyRule(state, n.aMap[cPrime], c)
	// now we must add c to the map of {a}
	containedIn := n.aMap[cPrime]
	if len(containedIn) == 0 {
		containedIn = make(map[uint]struct{}, 10)
		n.aMap[cPrime] = containedIn
	}
	containedIn[cPrime] = struct{}{}
	return result
}
