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

type SolverState struct {
	S []*BCSet
	R []*Relation

	sMutex []*sync.RWMutex
	rMutex []*sync.RWMutex
}

func NewSolverState(c *ELBaseComponents) *SolverState {
	res := SolverState{
		S:      nil,
		R:      nil,
		sMutex: nil,
		rMutex: nil,
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
	state.rMutex[r].Lock()
	res := state.R[r].Add(c, d)
	state.rMutex[r].Unlock()
	return res
}

type SNotification interface {
	// Information, that C' was added to S(C)
	GetSNotification(state *SolverState, c, cPrime uint) bool
}

type RNotification interface {
	// Information, that (C, D) was added to R(r)
	GetRNotification(state *SolverState, r, c, d uint) bool
}

// uint stands for the D in the rule
type CR1Notification uint

func (n CR1Notification) GetSNotification(state *SolverState, c, cPrime uint) bool {
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

func (n *CR2Notfiaction) GetSNotification(state *SolverState, c, cPrime uint) bool {
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

func (n CR3Notification) GetSNotification(state *SolverState, c, cPrime uint) bool {
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

func (n *CR4Notification) updateLHS(state *SolverState) bool {
	// iterate over each C s.t. (C, D) is in R(r)
	// lock relation r for reading
	state.RLockRole(n.r)
	defer state.RUnlockRole(n.r)
	dMap, hasMap := state.R[n.r].reverseMapping[n.d]
	if !hasMap {
		return false
	}
	// iterate over each c in dMap
	res := false
	for c, _ := range dMap {
		res = state.AddConcept(c, n.e) || res
	}
	return res
}

func (n *CR4Notification) GetSNotification(state *SolverState, d, dPrime uint) bool {
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

func (n *CR4Notification) GetRNotification(state *SolverState, r, c, d uint) bool {
	n.mutex.Lock()
	// apply only if the second part of the rule is already fulfilled
	// in this case simply apply the rule
	res := n.containsConcept && state.AddConcept(c, n.e)
	n.mutex.Unlock()
	return res
}

type CR5Notification struct {
	c, d, r     uint
	containsBot bool
	mutex       *sync.Mutex
}

func NewCR5Notification(c, d, r uint) *CR5Notification {
	var mutex sync.Mutex
	return &CR5Notification{
		c:           c,
		d:           d,
		r:           r,
		containsBot: false,
		mutex:       &mutex,
	}
}

func (n *CR5Notification) updateLHS(state *SolverState) bool {
	// iterate over each C s.t. (C, D) is in R(r)
	// lock relation r for reading
	state.RLockRole(n.r)
	defer state.RUnlockRole(n.r)
	dMap, hasMap := state.R[n.r].reverseMapping[n.d]
	if !hasMap {
		return false
	}
	// iterate over each c in dMap
	res := false
	for c, _ := range dMap {
		// TODO is zero correct? should be...
		res = state.AddConcept(c, 0) || res
	}
	return res
}

func (n *CR5Notification) GetSNotification(state *SolverState, d, bot uint) bool {
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

func (n *CR5Notification) GetRNotification(state *SolverState, r, c, d uint) bool {
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

func (n CR10Notification) GetRNotification(state *SolverState, r, c, d uint) bool {
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

func (n CR11Notification) GetRNotification(state *SolverState, r, c, d uint) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	switch r {
	case n.r1:
		// iterate over each (D, E)
		state.RLockRole(n.r2)
		defer state.RUnlockRole(n.r2)
		succMap, hasMap := state.R[n.r2].mapping[d]
		if !hasMap {
			return false
		}
		result := false
		for e, _ := range succMap {
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
		predMap, hasMap := state.R[n.r1].reverseMapping[d]
		if !hasMap {
			return false
		}
		result := false
		for c, _ := range predMap {
			result = state.AddRole(n.r3, c, e) || result
		}
		return result
	default:
		log.Printf("Invalid notification for rule CR11: Only waiting for changes on %d and %d, got %d",
			n.r1, n.r2, r)
		return false
	}
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
			values[onValue] = make([]SNotification, 10)
		}
		values[onValue] = append(values[onValue], n)
	}

	addR := func(source uint, n RNotification) {
		rMutex.Lock()
		defer rMutex.Unlock()
		rm.rRules[source] = append(rm.rRules[source], n)
	}

	wg.Add(4)

	// Normalized CIs
	go func() {
		defer wg.Done()
		for _, gci := range tbox.CIs {
			if gci.C2 == nil {
				// rule CR1
				cPrime := gci.C1.NormalizedID(components)
				d := gci.D.NormalizedID(components)
				// TODO is this a good idea? hmmm...
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
				// TODO again, good idea?
				n := NewCR2Notfication(c1, c2, d)
				var c uint = 1
				for ; c < numBCD; c++ {
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
	wg.Wait()
}
