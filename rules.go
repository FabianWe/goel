// The MIT License (MIT)
//
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
	"log"
	"sync"
)

// StateHandler is used by the concurrent (and maybe other solvers) to update
// state information. That is update the mappings S(C) and R(R) and check if
// certain elements are present in it.
// For R(r) it is sometimes required to iterate over each element in R(r),
// therefor exists methods that take a channel as en input, write each element
// to that channel and then close the channel. Thus methods that require to
// iterate over each R(r) start a go routine with a channel and iterate the
// channel until it is closed.
// Also since sometimes direct read / write operations are required we must be
// able to (read-)lock a certain S(C) or R(r).
// A basic implementation is given in SolverState, other state handlers or even
// solvers can simply delegate certain methods to this basic implementation.
// That is they can add additional logic (such as triggering certain rules)
// and let the SolverState handle everything else. Of course it is also
// possible to write a completely new handler.
//
// State handlers must be safe to use concurrently from multiple go routines.
type StateHandler interface {
	// Methods for locking / unlocking (read)-access.

	// RLockConcept locks S(C) for reading.
	RLockConcept(c uint)

	// RUnlockConcept unlocks S(C) reading access.
	RUnlockConcept(c uint)

	// LockConcept locks S(C) for reading / writing.
	LockConcept(c uint)

	// UnlockConcept unlocks S(C) reading / writing access.
	UnlockConcept(c uint)

	// RLockRole locks R(r) for reading.
	RLockRole(r uint)

	// RUnlockRole unlocks R(r) reading access.
	RUnlockRole(r uint)

	// LockRole locks R(r) for reading / writing.
	LockRole(r uint)

	// UnlockRole unlocks R(r) reading / writing access.
	UnlockRole(r uint)

	// Methods for changing / reading S(C) and R(r).

	// ContainsConcept checks whether D ∈ S(D).
	ContainsConcept(c, d uint) bool

	// AddConcept adds D to S(C) and returns true if the update changed S(C).
	AddConcept(c, d uint) bool

	// UnionConcepts adds all elements from S(D) to S(C), thus performs the update
	// S(C) = S(C) ∪ S(D). Returns true if some elements were added to S(C).
	UnionConcepts(c, d uint) bool

	// ContainsRole checks whether (C, D) ∈ R(r).
	ContainsRole(r, c, d uint) bool

	// AddRole adds (C, D) to R(r). It must also update the graph.
	AddRole(r, c, d uint) bool

	// RoleMapping returns all pairs (C, D) in R(r) for a given C.
	RoleMapping(r, c uint, ch chan<- uint)

	// ReverseRoleMapping returns all pairs (C, D) in R(r) for a given D.
	ReverseRoleMapping(r, d uint, ch chan<- uint)

	// RLockGraph locks the graph for reading.
	RLockGraph()

	// RUnlockGraph unlocks the graph reading access.
	RUnlockGraph()

	// LockGraph locks the graph for reading / writing.
	LockGraph()

	// UnlockGraph unlocks the graph reading / writing access.
	UnlockGraph()

	// GetGraph returns the graph, if you wish to read / write use the lock methods
	// first.
	GetGraph() ConceptGraph
}

// SolverState is an implementation of StateHandler, for more details see there.
// It protects each S(C) and R(r) with a RWMutex.
type SolverState struct {
	S []*BCSet
	R []*Relation

	sMutex []*sync.RWMutex
	rMutex []*sync.RWMutex

	graph      ConceptGraph
	graphMutex *sync.RWMutex

	extendedSearch ExtendedReachabilitySearch
	search         ReachabilitySearch
	searcher       *ExtendedGraphSearcher
}

// NewSolverState returns a new solver state given the base components,
// thus it initializes S and R and the mutexes used to control r/w access.
func NewSolverState(c *ELBaseComponents, graph ConceptGraph,
	extendedSearch ExtendedReachabilitySearch, search ReachabilitySearch) *SolverState {
	var graphMutex sync.RWMutex
	res := SolverState{
		S:              nil,
		R:              nil,
		sMutex:         nil,
		rMutex:         nil,
		graph:          graph,
		graphMutex:     &graphMutex,
		extendedSearch: extendedSearch,
		search:         search,
		searcher:       nil,
	}
	// initialize S and R concurrently
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	// also initialize graph and the graph searcher
	numBCD := c.NumBCD() + 1
	var wg sync.WaitGroup
	wg.Add(4)
	// initialize S
	go func() {
		res.S = make([]*BCSet, numBCD)
		res.sMutex = make([]*sync.RWMutex, numBCD)
		var i uint = 1
		for ; i < numBCD; i++ {
			res.S[i] = NewBCSet(c, 10)
			var m sync.RWMutex
			res.sMutex[i] = &m
		}
		wg.Done()
	}()
	// initialize R
	go func() {
		res.R = make([]*Relation, c.Roles)
		res.rMutex = make([]*sync.RWMutex, c.Roles)
		var i uint = 0
		for ; i < c.Roles; i++ {
			res.R[i] = NewRelation(10)
			var m sync.RWMutex
			res.rMutex[i] = &m
		}
		wg.Done()
	}()
	// graph
	go func() {
		res.graph.Init(numBCD)
		wg.Done()
	}()
	// graph searcher
	go func() {
		res.searcher = NewExtendedGraphSearcher(res.extendedSearch, res.search, c)
		wg.Done()
	}()
	wg.Wait()
	return &res
}

func (state *SolverState) RLockConcept(c uint) {
	state.sMutex[c].RLock()
}

func (state *SolverState) RUnlockConcept(c uint) {
	state.sMutex[c].RUnlock()
}

func (state *SolverState) LockConcept(c uint) {
	state.sMutex[c].Lock()
}

func (state *SolverState) UnlockConcept(c uint) {
	state.sMutex[c].Unlock()
}

func (state *SolverState) RLockRole(r uint) {
	state.rMutex[r].RLock()
}

func (state *SolverState) RUnlockRole(r uint) {
	state.rMutex[r].RUnlock()
}

func (state *SolverState) LockRole(r uint) {
	state.rMutex[r].Lock()
}

func (state *SolverState) UnlockRole(r uint) {
	state.rMutex[r].Unlock()
}

// Does S(C) contain D?
func (state *SolverState) ContainsConcept(c, d uint) bool {
	state.sMutex[c].RLock()
	res := state.S[c].ContainsID(d)
	state.sMutex[c].RUnlock()
	return res
}

// add D to S(C)
func (state *SolverState) AddConcept(c, d uint) bool {
	state.sMutex[c].Lock()
	res := state.S[c].AddID(d)
	state.sMutex[c].Unlock()
	return res
}

// S(C) = S(C) + S(D)
func (state *SolverState) UnionConcepts(c, d uint) bool {
	state.sMutex[c].Lock()
	res := state.S[c].Union(state.S[d])
	state.sMutex[c].Unlock()
	return res
}

func (state *SolverState) ContainsRole(r, c, d uint) bool {
	state.rMutex[r].RLock()
	res := state.R[r].Contains(c, d)
	state.rMutex[r].RUnlock()
	return res
}

func (state *SolverState) AddRole(r, c, d uint) bool {
	// we must update the graph as well, both methods may look do to the blocking
	// mechanisms, so why not do it concurrently
	// TODO
	state.rMutex[r].Lock()
	res := state.R[r].Add(c, d)
	state.rMutex[r].Unlock()
	return res
}

func (state *SolverState) RoleMapping(r, c uint, ch chan<- uint) {
	m := state.R[r].mapping[c]
	for d, _ := range m {
		ch <- d
	}
	close(ch)
}

func (state *SolverState) ReverseRoleMapping(r, d uint, ch chan<- uint) {
	m := state.R[r].reverseMapping[d]
	for c, _ := range m {
		ch <- c
	}
	close(ch)
}

func (state *SolverState) RLockGraph() {
	state.graphMutex.RLock()
}

func (state *SolverState) RUnlockGraph() {
	state.graphMutex.RUnlock()
}

func (state *SolverState) LockGraph() {
	state.graphMutex.Lock()
}

func (state *SolverState) UnlockGraph() {
	state.graphMutex.Unlock()
}

func (state *SolverState) GetGraph() ConceptGraph {
	return state.graph
}

type RuleMap struct {
	sRules []map[uint][]SNotification
	rRules [][]RNotification
}

func NewRuleMap() *RuleMap {
	return &RuleMap{
		sRules: nil,
		rRules: nil,
	}
}

// TODO 5 and 6 still missing
func (rm *RuleMap) Init(tbox *NormalizedTBox) {
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	components := tbox.Components
	numBCD := components.NumBCD() + 1

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		rm.sRules = make([]map[uint][]SNotification, numBCD)
		var i uint = 1
		for ; i < numBCD; i++ {
			rm.sRules[i] = make(map[uint][]SNotification, 10)
		}
	}()
	go func() {
		defer wg.Done()
		rm.rRules = make([][]RNotification, components.Roles)
		var i uint = 0
		for ; i < components.Roles; i++ {
			rm.rRules[i] = make([]RNotification, 0, 10)
		}
	}()
	wg.Wait()

	// for all different "forms" we run a go routine
	// to create rules
	// to make things easy we use two mutexes to control s and r
	var sMutex, rMutex sync.Mutex

	addS := func(source, onValue uint, n SNotification) {
		sMutex.Lock()
		defer sMutex.Unlock()
		values := rm.sRules[source]
		_, hasM := values[onValue]
		if !hasM {
			values[onValue] = make([]SNotification, 0, 10)
		}
		values[onValue] = append(values[onValue], n)
	}

	addR := func(source uint, n RNotification) {
		rMutex.Lock()
		defer rMutex.Unlock()
		rm.rRules[source] = append(rm.rRules[source], n)
	}

	wg.Add(5)

	// Normalized CIs
	go func() {
		defer wg.Done()
		for _, gci := range tbox.CIs {
			if gci.C2 == nil {
				// rule CR1
				cPrime := gci.C1.NormalizedID(components)
				d := gci.D.NormalizedID(components)
				// TODO is this a good idea? hmmm... Seems to work here.
				n := NewCR1Notification(d)
				// add a rule for each possible C
				var c uint = 1
				for ; c < numBCD; c++ {
					// add rule that waits on an update on C (for D getting added)
					addS(c, cPrime, n)
				}
			} else {
				// rule CR2
				c1 := gci.C1.NormalizedID(components)
				c2 := gci.C2.NormalizedID(components)
				d := gci.D.NormalizedID(components)
				// add a rule for each possible C
				var c uint = 1
				for ; c < numBCD; c++ {
					n := NewCR2Notfication(c1, c2, d)
					// add rule that waits for an update on C and then adds D
					addS(c, c1, n)
					addS(c, c2, n)
				}
			}
		}
	}()

	// NormalizedCIRightEx
	go func() {
		defer wg.Done()
		for _, ex := range tbox.CIRight {
			cPrime := ex.C1.NormalizedID(components)
			r := uint(ex.R)
			d := ex.C2.NormalizedID(components)
			n := NewCR3Notification(r, d)
			var c uint = 1
			for ; c < numBCD; c++ {
				addS(c, cPrime, n)
			}
		}
	}()

	// NormalizedCILeftEx
	go func() {
		defer wg.Done()

		for _, ex := range tbox.CILeft {
			r := uint(ex.R)
			dPrime := ex.C1.NormalizedID(components)
			e := ex.D.NormalizedID(components)
			var d uint = 1
			for ; d < numBCD; d++ {
				n := NewCR4Notification(r, dPrime, e, d)
				addS(d, dPrime, n)
				addR(r, n)
			}
		}
	}()

	// NormalizedRI
	go func() {
		defer wg.Done()

		for _, ri := range tbox.RIs {

			if ri.R2 == NoRole {
				// CR10
				r := uint(ri.R1)
				s := uint(ri.S)
				n := NewCR10Notification(s)
				addR(r, n)
			} else {
				// CR11
				r1 := uint(ri.R1)
				r2 := uint(ri.R2)
				r3 := uint(ri.S)
				n := NewCR11Notification(r1, r2, r3)
				addR(r1, n)
				addR(r2, n)
			}
		}
	}()

	// CR5
	go func() {
		defer wg.Done()
		// for each d add a notification, also add it to each r
		var d uint = 1
		for ; d < numBCD; d++ {
			var r uint = 0
			for ; r < components.Roles; r++ {
				n := NewCR5Notification(d, r)
				addS(d, 0, n)
				addR(r, n)
			}
		}
	}()

	wg.Wait()
}

type SNotification interface {
	// Information, that C' was added to S(C)
	GetSNotification(state StateHandler, c, cPrime uint) bool
}

type RNotification interface {
	// Information, that (C, D) was added to R(r)
	GetRNotification(state StateHandler, r, c, d uint) bool
}

// uint stands for the D in the rule
type CR1Notification uint

func (n CR1Notification) GetSNotification(state StateHandler, c, cPrime uint) bool {
	// we have to add D (from the rule) to S(C)
	return state.AddConcept(c, uint(n))
}

func NewCR1Notification(d uint) CR1Notification {
	return CR1Notification(d)
}

// uint stands for the D in the rule
type CR2Notfiaction struct {
	c1, c2           uint
	d                uint
	c1Found, c2Found bool
	mutex            *sync.Mutex
}

func NewCR2Notfication(c1, c2, d uint) *CR2Notfiaction {
	var m sync.Mutex
	return &CR2Notfiaction{
		c1:      c1,
		c2:      c2,
		d:       d,
		c1Found: false,
		c2Found: false,
		mutex:   &m,
	}
}

func (n *CR2Notfiaction) GetSNotification(state StateHandler, c, cPrime uint) bool {
	// first of all lock the mutex
	n.mutex.Lock()
	defer n.mutex.Unlock()
	switch cPrime {
	case n.c1:
		n.c1Found = true
	case n.c2:
		n.c2Found = true
	default:
		// TODO remove once tested
		log.Printf("Wrong handler for rule CR2: Got update for %d, but only listening to %d and %d",
			cPrime, n.c1, n.c2)
		return false
	}
	return n.c1Found && n.c2Found && state.AddConcept(c, n.d)
}

type CR3Notification struct {
	r, d uint
}

func NewCR3Notification(r, d uint) CR3Notification {
	return CR3Notification{
		r: r,
		d: d,
	}
}

func (n CR3Notification) GetSNotification(state StateHandler, c, cPrime uint) bool {
	return state.AddRole(n.r, c, n.d)
}

type CR4Notification struct {
	r, dPrime, e, d uint
	containsConcept bool
	mutex           *sync.Mutex
}

func NewCR4Notification(r, dPrime, e, d uint) *CR4Notification {
	var mutex sync.Mutex
	return &CR4Notification{
		r:               r,
		dPrime:          dPrime,
		e:               e,
		d:               d,
		containsConcept: false,
		mutex:           &mutex,
	}
}

func (n *CR4Notification) updateLHS(state StateHandler) bool {
	// iterate over each C s.t. (C, D) is in R(r)
	// lock relation r for reading
	state.RLockRole(n.r)
	defer state.RUnlockRole(n.r)
	// get reverse mapping for d
	ch := make(chan uint, 1)
	go state.ReverseRoleMapping(n.r, n.d, ch)
	// iterate over each c in dMap
	res := false
	for c := range ch {
		res = state.AddConcept(c, n.e) || res
	}
	return res
}

func (n *CR4Notification) GetSNotification(state StateHandler, d, dPrime uint) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// TODO remove once tested
	if d != n.d || dPrime != n.dPrime {
		log.Printf("Error in rule CR4: d != n.d or d' != n.d', got %d, %d, %d, %d",
			d, n.d, dPrime, n.dPrime)
		return false
	}
	// set contains concept to true, also apply the LHS rule
	// TODO remove once tested
	if n.containsConcept {
		log.Printf("Error in rule CR4: Concept rule applied twice")
	}
	n.containsConcept = true
	return n.updateLHS(state)
}

func (n *CR4Notification) GetRNotification(state StateHandler, r, c, d uint) bool {
	n.mutex.Lock()
	// apply only if the second part of the rule is already fulfilled
	// in this case simply apply the rule
	res := n.containsConcept && state.AddConcept(c, n.e)
	n.mutex.Unlock()
	return res
}

type CR5Notification struct {
	d, r        uint
	containsBot bool
	mutex       *sync.Mutex
}

func NewCR5Notification(d, r uint) *CR5Notification {
	var mutex sync.Mutex
	return &CR5Notification{
		d:           d,
		r:           r,
		containsBot: false,
		mutex:       &mutex,
	}
}

func (n *CR5Notification) updateLHS(state StateHandler) bool {
	// iterate over each C s.t. (C, D) is in R(r)
	// lock relation r for reading
	state.RLockRole(n.r)
	defer state.RUnlockRole(n.r)
	ch := make(chan uint, 1)
	go state.ReverseRoleMapping(n.r, n.d, ch)
	// iterate over each c in dMap
	res := false
	for c := range ch {
		// TODO is zero correct? should be...
		res = state.AddConcept(c, 0) || res
	}
	return res
}

func (n *CR5Notification) GetSNotification(state StateHandler, d, bot uint) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// TODO remove once tested
	if bot != 0 {
		log.Printf("Error in rule CR5: Expected bottom concept, but got %d", bot)
		return false
	}
	// set contains concept to true, also apply the LHS rule
	// TODO remove once tested
	if n.containsBot {
		log.Printf("Error in rule CR5: Concept rule applied twice")
	}
	n.containsBot = true
	return n.updateLHS(state)
}

func (n *CR5Notification) GetRNotification(state StateHandler, r, c, d uint) bool {
	n.mutex.Lock()
	// apply only if the second part of the rule is already fulfilled
	// in this case simply apply the rule
	res := n.containsBot && state.AddConcept(c, 0)
	n.mutex.Unlock()
	return res
}

// TODO CR6

// uint for the s
type CR10Notification uint

func NewCR10Notification(s uint) CR10Notification {
	return CR10Notification(s)
}

func (n CR10Notification) GetRNotification(state StateHandler, r, c, d uint) bool {
	return state.AddRole(uint(n), c, d)
}

type CR11Notification struct {
	r1, r2, r3 uint
	mutex      *sync.Mutex
}

func NewCR11Notification(r1, r2, r3 uint) CR11Notification {
	var mutex sync.Mutex
	return CR11Notification{
		r1:    r1,
		r2:    r2,
		r3:    r3,
		mutex: &mutex,
	}
}

func (n CR11Notification) GetRNotification(state StateHandler, r, c, d uint) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	switch r {
	case n.r1:
		// iterate over each (D, E)
		state.RLockRole(n.r2)
		defer state.RUnlockRole(n.r2)
		ch := make(chan uint, 1)
		go state.RoleMapping(n.r2, d, ch)
		result := false
		for e := range ch {
			result = state.AddRole(n.r3, c, e) || result
		}
		return result
	case n.r2:
		// first some renaming to keep it more readable...
		// in this case the names in the rule are (D, E) for R(r2)
		e := d
		d = c
		// iterate over each (C, D)
		// that is itertae over the reversed map of D
		state.RLockRole(n.r1)
		defer state.RUnlockRole(n.r1)
		ch := make(chan uint, 1)
		go state.ReverseRoleMapping(n.r1, d, ch)
		result := false
		for c := range ch {
			result = state.AddRole(n.r3, c, e) || result
		}
		return result
	default:
		log.Printf("Invalid notification for rule CR11: Only waiting for changes on %d and %d, got %d",
			n.r1, n.r2, r)
		return false
	}
}
