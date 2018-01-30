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
	"sync"
)

type sUpdate struct {
	c, d uint
}

func newSUpdate(c, d uint) sUpdate {
	return sUpdate{
		c: c,
		d: d,
	}
}

type rUpdate struct {
	r, c, d uint
}

func newRUpdate(r, c, d uint) rUpdate {
	return rUpdate{
		r: r,
		c: c,
		d: d,
	}
}

type RuleSolver struct {
	*SolverState
	*RuleMap

	pendingSUpdates []sUpdate
	pendingRUpdates []rUpdate
}

func NewRuleSolver() *RuleSolver {
	return &RuleSolver{nil, nil, nil, nil}
}

func (solver *RuleSolver) Init(tbox *NormalizedTBox) {
	solver.pendingSUpdates = make([]sUpdate, 0, 10)
	solver.pendingRUpdates = make([]rUpdate, 0, 10)
	// initialize the state and the rules
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		solver.SolverState = NewSolverState(tbox.Components)
		wg.Done()
	}()
	go func() {
		solver.RuleMap = NewRuleMap()
		solver.RuleMap.Init(tbox)
		wg.Done()
	}()
	wg.Wait()
}

func (solver *RuleSolver) AddConcept(c, d uint) bool {
	res := solver.SolverState.AddConcept(c, d)
	// TODO correct to do that here?
	if res {
		// add pending add
		update := newSUpdate(c, d)
		solver.pendingSUpdates = append(solver.pendingSUpdates, update)
	}
	return res
}

func (solver *RuleSolver) UnionConcepts(c, d uint) bool {
	// we don't want to iterate over each concept twice (once in the set union
	// and once here) so we simply do this by hand... Bit of code duplication
	// but I guess that's okay
	sc := solver.S[c].m
	sd := solver.S[d].m
	added := false
	for v, _ := range sd {
		// add to S(C)
		oldLen := len(sc)
		sc[v] = struct{}{}
		changed := oldLen != len(sc)
		if changed {
			added = true
			// add update
			// TODO again correct here?
			update := newSUpdate(c, v)
			solver.pendingSUpdates = append(solver.pendingSUpdates, update)
		}
	}
	return added
}

func (solver *RuleSolver) AddRole(r, c, d uint) bool {
	res := solver.SolverState.AddRole(r, c, d)
	// TODO again, is this the right place?
	if res {
		// add pending update
		update := newRUpdate(r, c, d)
		solver.pendingRUpdates = append(solver.pendingRUpdates, update)
	}
	return res
}

func (solver *RuleSolver) Solve(tbox *NormalizedTBox) {
	// TODO call init here, made this easier for testing during debuging.

	components := tbox.Components
	// add all initial setup steps, that is for each C add ⊤ and C to S(C).
	// that is for ⊤ add only ⊤, for all other C add ⊤ and C
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
		case len(solver.pendingSUpdates) != 0:
			// get next s update and apply it
			n := len(solver.pendingSUpdates)
			next := solver.pendingSUpdates[n-1]
			solver.pendingSUpdates = solver.pendingSUpdates[:n-1]
			// notify about the update
			c, d := next.c, next.d
			cRules := solver.RuleMap.sRules[c]
			notifications := cRules[d]
			for _, notification := range notifications {
				notification.GetSNotification(solver, c, d)
			}
			// once D has been added to D we can remove all the rules for C concerning D,
			// we never have to look at them again and we may release some memory
			delete(solver.RuleMap.sRules[c], d)
		case len(solver.pendingRUpdates) != 0:
			n := len(solver.pendingRUpdates)
			next := solver.pendingRUpdates[n-1]
			solver.pendingRUpdates = solver.pendingRUpdates[:n-1]
			r, c, d := next.r, next.c, next.d
			notifications := solver.RuleMap.rRules[next.r]
			for _, notification := range notifications {
				notification.GetRNotification(solver, r, c, d)
			}
		default:
			break L
		}
	}
}
