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

import (
	"fmt"
	"sync"
)

// SUpdate is a type that stores the information that D has been added to S(C).
// It is usually used in a queue that stores all updates that still must be
// executed (notifications for that update must be issued).
// TODO Is there a mix-up with C / D?
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

	SubsetConcepts(c, d uint) bool
	// UpdateGraph(c, d uint) bool
	// TODO add search method(s) here.
	ExtendedSearch(goals map[uint]struct{}, additionalStart uint) map[uint]struct{}

	BidrectionalSearch(oldElements map[uint]struct{}, newElement uint) map[uint]BidirectionalSearch
	// TODO describe requirements
	AddSubsetRule(c, d uint, ch <-chan bool) bool
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
	// TODO remove print
	// fmt.Println(state.Graph)
	res := state.Searcher.Search(state.Graph, goals, additionalStart)
	state.graphMutex.RUnlock()
	return res
}

func (state *AllChangesSolverState) BidrectionalSearch(oldElements map[uint]struct{},
	newElement uint) map[uint]BidirectionalSearch {
	state.graphMutex.RLock()
	// TODO remove print
	// fmt.Println(state.Graph)
	res := state.Searcher.BidrectionalSearch(state.Graph, oldElements, newElement)
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

type AllGraphChangeNotification interface {
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

func (n *AllChangesCR6) applyRuleBidirectional(state AllChangesState, goals map[uint]struct{}, c uint) bool {
	// TODO again, is filtering correct?
	filtered := make(map[uint]struct{}, len(goals))
	for d, _ := range goals {
		// TODO correct?
		if !state.SubsetConcepts(d, c) || !state.SubsetConcepts(c, d) {
			filtered[d] = struct{}{}
		}
	}
	if len(filtered) == 0 {
		return false
	}
	// fmt.Println("Searching bidirectional to and from", StringUintSet(filtered), "and", c)
	connected := state.BidrectionalSearch(filtered, c)
	// TODO remove prints
	// fmt.Println("Connected", connected)

	result := false
	for d, connType := range connected {
		if c == d {
			continue
		}
		switch connType {
		case BidrectionalDirect:
			done := make(chan bool, 1)
			state.AddSubsetRule(c, d, done)
			result = state.UnionConcepts(c, d) || result
			done <- true
		case BidrectionalReverse:
			done := make(chan bool, 1)
			state.AddSubsetRule(d, c, done)
			// fmt.Printf("Union of: S(%d) = S(%d) U S(%d)\n", d, d, c)
			result = state.UnionConcepts(d, c) || result
			done <- true
		case BidrectionalBoth:
			// TODO is this still correct in the concurrent version?
			done1 := make(chan bool, 1)
			state.AddSubsetRule(c, d, done1)
			result = state.UnionConcepts(c, d) || result
			done1 <- true
			done2 := make(chan bool, 1)
			state.AddSubsetRule(d, c, done2)
			result = state.UnionConcepts(d, c) || result
			done2 <- true
		}
	}
	return result
}

func (n *AllChangesCR6) applyRuleDirectOnly(state AllChangesState, goals map[uint]struct{}, c uint) bool {
	// before doing a search on the graph reduce the number of goals by checking
	// the subset property this might help us to speed up the search
	// TODO is this correct even in a concurrent version?

	filtered := make(map[uint]struct{}, len(goals))
	for d, _ := range goals {
		if !state.SubsetConcepts(d, c) {
			filtered[d] = struct{}{}
		}
	}
	if len(filtered) == 0 {
		return false
	}
	connected := state.ExtendedSearch(filtered, c)
	// TODO remove prints
	// fmt.Println("Searching to", filtered, "from", c)
	// fmt.Println("Connected:", connected)
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
			result = n.applyRuleDirectOnly(state, containedIn, c) || result
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
	// TODO update documentation
	result := n.applyRuleBidirectional(state, n.aMap[cPrime], c)
	// now we must add C to the map of {a}
	containedIn := n.aMap[cPrime]
	if len(containedIn) == 0 {
		containedIn = make(map[uint]struct{}, 10)
		n.aMap[cPrime] = containedIn
	}
	containedIn[c] = struct{}{}
	return result
}

// AllChangesRuleMap is an extension of RuleMap. It has some extended
// functionality: It stores the subset mapping as required by rule CR6
// and methods to add new elements to it / perform the update on a given state.
// These functions are safe for concurrent use (protected by a mutex, so better
// understand what happens to avoid deadlocks; sorry).
// And also holds an instance of AllChangesCR6 to perform this update when
// required.
type AllChangesRuleMap struct {
	*RuleMap

	// additional mapping that stores which subset relations must be maintained,
	// that is rule CR6 forces us to take care that (if a certain condition
	// is true) S(C) must always be a subset of S(D).
	// So whenever C' gets added to S(D) we must add it to S(C) as well.
	// This maps stores for each D all C for which an update on S(D) triggers an
	// update on S(C).
	subsetMap map[uint]map[uint]struct{}

	// a mutex used to control access on subsetMap
	// for simplicity we don't use a mutex for each concept C in the map but
	// just one that controls the whole map
	subsetMutex *sync.RWMutex

	// An instance of CR6 to be executed whenever S(C) changes (for any C)
	// or the graph is changed, no interfaces here, they're just there for
	// clarification
	cr6 *AllChangesCR6
}

func NewAllChangesRuleMap() *AllChangesRuleMap {
	var m sync.RWMutex
	subsetMap := make(map[uint]map[uint]struct{})
	cr6 := NewAllChangesCR6()
	return &AllChangesRuleMap{
		RuleMap:     NewRuleMap(),
		subsetMap:   subsetMap,
		subsetMutex: &m,
		cr6:         cr6,
	}
}

func (rm *AllChangesRuleMap) Init(tbox *NormalizedTBox) {
	rm.RuleMap.Init(tbox)
}

func (rm *AllChangesRuleMap) ApplySubsetNotification(state AllChangesState, d, cPrime uint) bool {
	// lock mutex
	rm.subsetMutex.RLock()
	defer rm.subsetMutex.RUnlock()
	// iterate over each c in map[D]
	updates := rm.subsetMap[d]
	result := false
	for c, _ := range updates {
		if c == d {
			continue
		}
		// add C' to S(C)
		result = state.AddConcept(c, cPrime) || result
	}
	return result
}

func (rm *AllChangesRuleMap) newSubsetRule(c, d uint) bool {
	// lock mutex
	rm.subsetMutex.Lock()
	defer rm.subsetMutex.Unlock()
	// get map for d
	m := rm.subsetMap[d]
	if len(m) == 0 {
		m = make(map[uint]struct{})
		rm.subsetMap[d] = m
	}
	oldLen := len(m)
	m[c] = struct{}{}
	return oldLen != len(m)
}

type AllChangesSolver struct {
	*AllChangesSolverState
	*AllChangesRuleMap

	pendingSupdates []*SUpdate
	pendingRUpdates []*RUpdate
	graphChanged    bool

	// reequired for init later
	graph ConceptGraph

	search ExtendedReachabilitySearch
}

func NewAllChangesSolver(graph ConceptGraph, search ExtendedReachabilitySearch) *AllChangesSolver {
	if search == nil {
		search = BFSToSet
	}
	return &AllChangesSolver{
		AllChangesSolverState: nil,
		AllChangesRuleMap:     nil,
		pendingSupdates:       nil,
		pendingRUpdates:       nil,
		graphChanged:          false,
		graph:                 graph,
		search:                search,
	}
}

func (solver *AllChangesSolver) Init(tbox *NormalizedTBox) {
	// create pending slices and reset graph changed
	solver.pendingSupdates = make([]*SUpdate, 0, 10)
	solver.pendingRUpdates = make([]*RUpdate, 0, 10)
	solver.graphChanged = false
	// inditialize state and rules (concurrently)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		solver.AllChangesSolverState = NewAllChangesSolverState(tbox.Components,
			solver.graph, solver.search)
		wg.Done()
	}()
	go func() {
		solver.AllChangesRuleMap = NewAllChangesRuleMap()
		solver.AllChangesRuleMap.Init(tbox)
		wg.Done()
	}()
	wg.Wait()
}

func (solver *AllChangesSolver) AddConcept(c, d uint) bool {
	res := solver.AllChangesSolverState.AddConcept(c, d)
	// TODO right place?! I guess so
	if res {
		// add pending update
		update := NewSUpdate(c, d)
		solver.pendingSupdates = append(solver.pendingSupdates, update)
	}
	return res
}

func (solver *AllChangesSolver) UnionConcepts(c, d uint) bool {
	// we don't want to iterate over each concept twice (once in the set union
	// and once here) so we simply do this by hand... Bit of code duplication
	// but I guess that's okay

	// first we want to avoid some deadlocks (if c == d nothing happens but we
	// can't read / write at the same time)
	if c == d {
		return false
	}
	solver.sMutex[c].Lock()
	solver.sMutex[d].RLock()
	sc := solver.S[c].M
	sd := solver.S[d].M
	added := false
	for v, _ := range sd {
		// add to S(C)
		oldLen := len(sc)
		sc[v] = struct{}{}
		if oldLen != len(sc) {
			// change took place, add pending update
			added = true
			// TODO again: right place?
			update := NewSUpdate(c, v)
			solver.pendingSupdates = append(solver.pendingSupdates, update)
		}
	}
	solver.sMutex[c].Unlock()
	solver.sMutex[d].RUnlock()
	return added
}

func (solver *AllChangesSolver) AddRole(r, c, d uint) bool {
	// in this case we have to update both: the relation r as well as the graph
	// and we have to add a pending update: one if R(r) has changed and one if
	// the graph has changed
	// first try to add to relation
	if r == 0 && c == 10 && d == 13 {
		fmt.Println("ADDED!!!")
	}
	res := solver.AllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue a pending update
		update := NewRUpdate(r, c, d)
		solver.pendingRUpdates = append(solver.pendingRUpdates, update)
		// change graph
		solver.graphMutex.Lock()
		defer solver.graphMutex.Unlock()
		graphUpdate := solver.Graph.AddEdge(c, d)
		// if update changed something notify about the update
		if graphUpdate {
			solver.graphChanged = true
		}
	}
	return res
}

func (solver *AllChangesSolver) AddSubsetRule(c, d uint, ch <-chan bool) bool {
	// TODO check here or in newSubsetRule if c == d to avoid infinite
	// chains of adds, is this possible in some other rules as well?!
	// no concurrency here, so nothing to worry about, just add the new rule
	res := solver.newSubsetRule(c, d)
	// we're not really interested in the ch channel because nothing runs
	// concurrently
	// usually we should wait here, but we can completely ignore the channel
	go func() {
		<-ch
	}()
	return res
}

func (solver *AllChangesSolver) Solve(tbox *NormalizedTBox) {
	// TODO call init here, made this easier for testing during debuging.
	// add all initial setup steps, that is for each C add ⊤ and C to S(C):
	// ⊤ add only ⊤, for all other C add ⊤ and C
	components := tbox.Components
	solver.AddConcept(1, 1)
	var c uint = 2
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	numBCD := components.NumBCD() + 1
	for ; c < numBCD; c++ {
		solver.AddConcept(c, 1)
		solver.AddConcept(c, c)
	}
	// while there are still pending updates apply those updates
L:
	for {
		switch {
		case len(solver.pendingSupdates) != 0:
			// get next s update and apply it
			n := len(solver.pendingSupdates)
			next := solver.pendingSupdates[n-1]
			// maybe help the garbage collection a bit if slice grows bigger and
			// bigger
			solver.pendingSupdates[n-1] = nil
			solver.pendingSupdates = solver.pendingSupdates[:n-1]
			// do notifications for that update
			c, d := next.C, next.D
			// first lookup all rules that are interested in an update
			// on S(D)
			// TODO here is a mixup?
			// or is just the documentation wrong?
			notifications := solver.SRules[d]
			// now iterate over each notification and apply it
			for _, notification := range notifications {
				notification.GetSNotification(solver, c, d)
			}
			// once the add is done we never have to worry about those rules again,
			// we will never apply them here again, so we can delete the entry
			// TODO may not be so wise, so I don't do it (if somehow we have to use
			// the rules again)
			// now also do a notification for CR6
			solver.cr6.GetSNotification(solver, c, d)
			// apply subset notifications for cr6
			// TODO correct? also looks very ugly...
			// TODO this might not be correct!
			// The problem is that we don't call all our update methods here!
			// we call them on AllChangesSolverState and this will not create
			// all events correctly!
			solver.AllChangesRuleMap.ApplySubsetNotification(solver, c, d)
		case len(solver.pendingRUpdates) != 0:
			// get next r update and apply it
			n := len(solver.pendingRUpdates)
			next := solver.pendingRUpdates[n-1]
			solver.pendingRUpdates[n-1] = nil
			solver.pendingRUpdates = solver.pendingRUpdates[:n-1]
			// do notifications for the update
			r, c, d := next.R, next.C, next.D
			// first all notifications waiting for r
			notifications := solver.RRules[r]
			for _, notification := range notifications {
				notification.GetRNotification(solver, r, c, d)
			}
			// now inform CR5 (or however else is waiting on an update on all roles)
			notifications = solver.RRules[uint(NoRole)]
			for _, notification := range notifications {
				notification.GetRNotification(solver, r, c, d)
			}
		case solver.graphChanged:
			solver.cr6.GetGraphNotification(solver)
			solver.graphChanged = false
		default:
			break L
		}
	}
}
