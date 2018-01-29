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

	sMutex *sync.RWMutex
	rMutex *sync.RWMutex
}

func NewSolverState(c *ELBaseComponents) *SolverState {
	var sMutex, rMutex sync.RWMutex
	res := SolverState{
		S:      nil,
		R:      nil,
		sMutex: &sMutex,
		rMutex: &rMutex,
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

// Does S(C) contain D?
func (state *SolverState) ContainsConcept(c, d uint) bool {
	state.sMutex.RLock()
	res := state.S[c].ContainsID(d)
	state.sMutex.RUnlock()
	return res
}

// add D to S(C)
func (state *SolverState) AddConcept(c, d uint) bool {
	state.sMutex.Lock()
	res := state.S[c].AddID(d)
	state.sMutex.Unlock()
	return res
}

// S(C) = S(C) + S(D)
func (state *SolverState) UnionConcepts(c, d uint) bool {
	state.sMutex.Lock()
	res := state.S[c].Union(state.S[d])
	state.sMutex.Unlock()
	return res
}

func (state *SolverState) ContainsRole(r, c, d uint) bool {
	state.rMutex.RLock()
	res := state.R[r].Contains(c, d)
	state.rMutex.RUnlock()
	return res
}

func (state *SolverState) AddRole(r, c, d uint) bool {
	state.rMutex.Lock()
	res := state.R[r].Add(c, d)
	state.rMutex.Unlock()
	return res
}

type SNotification interface {
	// Information, that C' was added to S(C)
	GetSNotification(state *SolverState, c, cPrime uint) bool
}

type RNotification interface {
	// Information, that (C, D) was added to R(r)
	GetRNotification(state *SolverState, r, c, d uint)
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
	return false
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

	return true
}
