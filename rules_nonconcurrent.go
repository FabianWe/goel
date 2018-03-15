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
	"sync"

	"github.com/FabianWe/goel/domains"
)

type NCSolverState struct {
	components *ELBaseComponents

	S []*BCSet
	R []*Relation

	domains *domains.CDManager
}

func NewNCSolverState(c *ELBaseComponents, domains *domains.CDManager) *NCSolverState {
	res := NCSolverState{
		components: c,
		S:          nil,
		R:          nil,
		domains:    domains,
	}
	// initialize S and R concurrently
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	numBCD := c.NumBCD() + 1
	var wg sync.WaitGroup
	wg.Add(2)
	// initialize S
	go func() {
		res.S = make([]*BCSet, numBCD)
		var i uint = 1
		for ; i < numBCD; i++ {
			res.S[i] = NewBCSet(c, 10)
		}
		wg.Done()
	}()
	// initialize R
	go func() {
		res.R = make([]*Relation, c.Roles)
		var i uint = 0
		for ; i < c.Roles; i++ {
			res.R[i] = NewRelation(10)
		}
		wg.Done()
	}()
	wg.Wait()
	return &res
}

func (state *NCSolverState) ContainsConcept(c, d uint) bool {
	return state.S[c].ContainsID(d)
}

func (state *NCSolverState) AddConcept(c, d uint) bool {
	return state.S[c].AddID(d)
}

func (state *NCSolverState) UnionConcepts(c, d uint) bool {
	if c == d {
		return false
	}
	return state.S[c].Union(state.S[d])
}

func (state *NCSolverState) ContainsRole(r, c, d uint) bool {
	return state.R[r].Contains(c, d)
}

func (state *NCSolverState) AddRole(r, c, d uint) bool {
	return state.R[r].Add(c, d)
}

func (state *NCSolverState) RoleMapping(r, c uint) []uint {

	m := state.R[r].Mapping[c]
	res := make([]uint, len(m))

	var i uint
	for d, _ := range m {
		res[i] = d
		i++
	}

	return res
}

func (state *NCSolverState) ReverseRoleMapping(r, d uint) []uint {

	m := state.R[r].ReverseMapping[d]
	res := make([]uint, len(m))

	var i uint
	for c, _ := range m {
		res[i] = c
		i++
	}
	return res
}

func (state *NCSolverState) SubsetConcepts(c, d uint) bool {
	return state.S[c].IsSubset(state.S[d])
}

func (state *NCSolverState) GetComponents() *ELBaseComponents {
	return state.components
}

func (state *NCSolverState) GetCDs() *domains.CDManager {
	return state.domains
}

func (state *NCSolverState) GetConjunction(c uint) [][]*domains.PredicateFormula {
	return state.S[c].GetCDConjunction(state.domains)
}

type NCAllChangesSolverState struct {
	// TODO graph still protected by mutex
	*NCSolverState

	Graph    ConceptGraph
	Searcher *ExtendedGraphSearcher
}

func NewNCAllChangesSolverState(c *ELBaseComponents,
	domains *domains.CDManager, g ConceptGraph, search ExtendedReachabilitySearch) *NCAllChangesSolverState {
	res := NCAllChangesSolverState{
		NCSolverState: nil,
		Graph:         g,
		Searcher:      nil,
	}
	// initalize SolverState, graph and searcher concurrently
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		res.NCSolverState = NewNCSolverState(c, domains)
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

func (state *NCAllChangesSolverState) ExtendedSearch(goals map[uint]struct{},
	additionalStart uint) map[uint]struct{} {
	return state.Searcher.Search(state.Graph, goals, additionalStart)
}

func (state *NCAllChangesSolverState) BidrectionalSearch(oldElements map[uint]struct{},
	newElement uint) map[uint]BidirectionalSearch {
	return state.Searcher.BidrectionalSearch(state.Graph, oldElements, newElement)
}

func (state *NCAllChangesSolverState) FindConnectedPairs(s map[uint]struct{}) *BCPairSet {
	return state.Searcher.FindConnectedPairs(state.Graph, s)
}

type NCRBSolver struct {
	*NCAllChangesSolverState
	*AllChangesRuleMap

	pendingSupdates []*SUpdate
	pendingRUpdates []*RUpdate
	graphChanged    bool

	// reequired for init later
	graph ConceptGraph

	search ExtendedReachabilitySearch

	// TODO new: I think because graph search runs concurrently the pendingRUpdates
	// must be protected as well!
	// ignored here at the moment
	// pendingRMutex *sync.Mutex
}

type NCAllChangesCR6 struct {
	aMap map[uint]map[uint]struct{}
}

func NewNCAllChangesCR6() *NCAllChangesCR6 {
	return &NCAllChangesCR6{aMap: make(map[uint]map[uint]struct{}, 10)}
}

func (n *NCAllChangesCR6) applyRuleBidirectional(state AllChangesState, goals map[uint]struct{}, c uint) bool {
	connected := state.BidrectionalSearch(goals, c)

	result := false
	for d, connType := range connected {
		if c == d {
			continue
		}
		switch connType {
		case BidrectionalDirect:
			state.AddSubsetRule(c, d)
			result = state.UnionConcepts(c, d) || result
		case BidrectionalReverse:
			state.AddSubsetRule(d, c)
			result = state.UnionConcepts(d, c) || result
		case BidrectionalBoth:
			state.AddSubsetRule(c, d)
			result = state.UnionConcepts(c, d) || result
			state.AddSubsetRule(d, c)
			result = state.UnionConcepts(d, c) || result
		}
	}
	return result
}

func (n *NCAllChangesCR6) runFindPairs(state AllChangesState, s map[uint]struct{}) bool {
	// call the state method to retrieve all connected pairs

	// TODO here some filtering might be useful, if already added a subset rule
	// it is not required to search again
	result := false
	pairs := state.FindConnectedPairs(s)
	// now add the rules
	for p, _ := range pairs.M {
		c, d := p.First, p.Second
		if c == d {
			continue
		}
		state.AddSubsetRule(c, d)
		result = state.UnionConcepts(c, d) || result
	}
	return result
}

func (n *NCAllChangesCR6) GetGraphNotification(state AllChangesState) bool {
	// if the graph has changed we iterate over all pairs and revaulate
	// the condition, that is we add new rules etc.
	// maybe there are nicer ways but we'll do the following:
	// iterate over each {a} and then perform the extended search for each C
	// that contains {a}.

	// removed here because there is no concurrency
	// lock mutex
	// n.aMutex.Lock()
	// defer n.aMutex.Unlock()
	result := false

	// here: no concurrency

	for _, containedIn := range n.aMap {
		result = n.runFindPairs(state, containedIn) || result
	}
	return result
}

func (n *NCAllChangesCR6) GetSNotification(state AllChangesState, c, cPrime uint) bool {
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
	// removed since there is no concurrency

	// n.aMutex.Lock()
	// defer n.aMutex.Unlock()

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

// A reimplementation of AllChangesRuleMap with no concurrency involved

type NCAllChangesRuleMap struct {
	*RuleMap

	// additional mapping that stores which subset relations must be maintained,
	// that is rule CR6 forces us to take care that (if a certain condition
	// is true) S(C) must always be a subset of S(D).
	// So whenever C' gets added to S(D) we must add it to S(C) as well.
	// This maps stores for each D all C for which an update on S(D) triggers an
	// update on S(C).
	subsetMap map[uint]map[uint]struct{}

	// An instance of CR6 to be executed whenever S(C) changes (for any C)
	// or the graph is changed, no interfaces here, they're just there for
	// clarification
	cr6 *NCAllChangesCR6
}

func NewNCAllChangesRuleMap() *NCAllChangesRuleMap {
	subsetMap := make(map[uint]map[uint]struct{})
	cr6 := NewNCAllChangesCR6()
	return &NCAllChangesRuleMap{
		RuleMap:   NewRuleMap(),
		subsetMap: subsetMap,
		cr6:       cr6,
	}
}

func (rm *NCAllChangesRuleMap) Init(tbox *NormalizedTBox) {
	rm.RuleMap.Init(tbox)
}

func (rm *NCAllChangesRuleMap) ApplySubsetNotification(state AllChangesState, d, cPrime uint) bool {
	// lock mutex
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

func (rm *NCAllChangesRuleMap) newSubsetRule(c, d uint) bool {
	if c == d {
		return false
	}
	// lock mutex
	// rm.subsetMutex.Lock()
	// defer rm.subsetMutex.Unlock()
	// no mutex required
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

func NewNCRBSolver(graph ConceptGraph, search ExtendedReachabilitySearch) *NCRBSolver {
	if search == nil {
		search = BFSToSet
	}
	return &NCRBSolver{
		NCAllChangesSolverState: nil,
		AllChangesRuleMap:       nil,
		pendingSupdates:         nil,
		pendingRUpdates:         nil,
		graphChanged:            false,
		graph:                   graph,
		search:                  search,
	}
}

func (solver *NCRBSolver) Init(tbox *NormalizedTBox, domains *domains.CDManager) {
	solver.pendingSupdates = make([]*SUpdate, 0, 10)
	solver.pendingRUpdates = make([]*RUpdate, 0, 10)
	solver.graphChanged = false
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		solver.NCAllChangesSolverState = NewNCAllChangesSolverState(tbox.Components,
			domains, solver.graph, solver.search)
		wg.Done()
	}()
	go func() {
		solver.AllChangesRuleMap = NewAllChangesRuleMap()
		solver.AllChangesRuleMap.Init(tbox)
		wg.Done()
	}()
	wg.Wait()
}

func (solver *NCRBSolver) AddConcept(c, d uint) bool {
	res := solver.NCAllChangesSolverState.AddConcept(c, d)
	if res {
		// add pending update
		update := NewSUpdate(c, d)
		solver.pendingSupdates = append(solver.pendingSupdates, update)
	}
	return res
}

func (solver *NCRBSolver) UnionConcepts(c, d uint) bool {
	// we don't want to iterate over each concept twice (once in the set union
	// and once here) so we simply do this by hand... Bit of code duplication
	// but I guess that's okay

	// first we want to avoid some deadlocks (if c == d nothing happens but we
	// can't read / write at the same time)
	if c == d {
		return false
	}

	// ugly duoMutex fix
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
			update := NewSUpdate(c, v)
			solver.pendingSupdates = append(solver.pendingSupdates, update)
		}
	}
	return added
}

func (solver *NCRBSolver) AddRole(r, c, d uint) bool {
	// in this case we have to update both: the relation r as well as the graph
	// and we have to add a pending update: one if R(r) has changed and one if
	// the graph has changed
	// first try to add to relation
	res := solver.NCAllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue a pending update
		update := NewRUpdate(r, c, d)
		// TODO new mutex, see above. Again ignored
		// solver.pendingRMutex.Lock()
		solver.pendingRUpdates = append(solver.pendingRUpdates, update)
		// change graph
		graphUpdate := solver.Graph.AddEdge(c, d)
		// if update changed something notify about the update
		if graphUpdate {
			solver.graphChanged = true
		}
		// TODO new mutex, see above
		// solver.pendingRMutex.Unlock()
	}
	return res
}

func (solver *NCRBSolver) AddSubsetRule(c, d uint) bool {
	// TODO check here or in newSubsetRule if c == d to avoid infinite
	// chains of adds, is this possible in some other rules as well?!
	// no concurrency here, so nothing to worry about, just add the new rule
	return solver.newSubsetRule(c, d)
}

func (solver *NCRBSolver) Solve(tbox *NormalizedTBox) {
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
			solver.AllChangesRuleMap.ApplySubsetNotification(solver, c, d)
			// apply notification for CR7/CR8
			solver.cr7A8.GetSNotification(solver, c, d)
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
			// TODO changed the position of graph changed, correct?
			solver.graphChanged = false
			solver.cr6.GetGraphNotification(solver)
		default:
			break L
		}
	}
}
