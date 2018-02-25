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

type ConcurrentNotificationSolver struct {
	*AllChangesSolver

	// mutexes to protec pending queues and graph changed variable
	rPendingMutex, sPendingMutex, graphChangedMutex *sync.Mutex
}

func NewConcurrentNotificationSolver(graph ConceptGraph, search ExtendedReachabilitySearch) *ConcurrentNotificationSolver {
	var rMutex, sMutex, graphChangedMutex sync.Mutex
	return &ConcurrentNotificationSolver{NewAllChangesSolver(graph, search), &rMutex, &sMutex, &graphChangedMutex}
}

// bit of code duplication here, but I think that's okay...

func (solver *ConcurrentNotificationSolver) AddConcept(c, d uint) bool {
	res := solver.AllChangesSolverState.AddConcept(c, d)
	if res {
		update := NewSUpdate(c, d)
		solver.sPendingMutex.Lock()
		solver.pendingSupdates = append(solver.pendingSupdates, update)
		solver.sPendingMutex.Unlock()
	}
	return res
}

func (solver *ConcurrentNotificationSolver) UnionConcepts(c, d uint) bool {
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

	// lock mutex only once, not every time in the for loop
	solver.sPendingMutex.Lock()

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

	solver.sMutex[c].Unlock()
	solver.sMutex[d].RUnlock()
	solver.sPendingMutex.Unlock()
	return added
}

func (solver *ConcurrentNotificationSolver) AddRole(r, c, d uint) bool {
	res := solver.AllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue pending updates
		update := NewRUpdate(r, c, d)
		solver.rPendingMutex.Lock()
		solver.pendingRUpdates = append(solver.pendingRUpdates, update)
		solver.rPendingMutex.Unlock()
		// update graph
		solver.graphMutex.Lock()
		graphUpdate := solver.Graph.AddEdge(c, d)
		solver.graphMutex.Unlock()
		// if update changed something notify about the update
		if graphUpdate {
			solver.graphChangedMutex.Lock()
			solver.graphChanged = true
			solver.graphChangedMutex.Unlock()
		}
	}
	return res
}

func (solver *ConcurrentNotificationSolver) AddSubsetRule(c, d uint, ch <-chan bool) bool {
	// this is exactly the same as in AllChangesSolver, but exists just to show
	// that we here don't have to worry about reads on the channel:
	// in this solver we add one update after another and do all notifications
	// concurrently.
	// that is: any changes made to some affected S(D) will apply the subset
	// update rule. And therefor we're safe to simply ignore the channel. The
	// channel is only important when also several updates happen concurrently

	res := solver.newSubsetRule(c, d)
	// not intersted in channel
	go func() {
		<-ch
	}()
	return res
}

func (solver *ConcurrentNotificationSolver) Solve(tbox *NormalizedTBox) {
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
			// get next update and apply all notifications concurrently
			n := len(solver.pendingSupdates)
			next := solver.pendingSupdates[n-1]
			// maybe help the garbage collector a bit
			solver.pendingSupdates[n-1] = nil
			solver.pendingSupdates = solver.pendingSupdates[:n-1]
			// do notifications
			c, d := next.C, next.D
			notifications := solver.SRules[d]
			// iterate over each notification and apply it, we use a waitgroup to
			// later wait for all updates to happen
			var wg sync.WaitGroup
			wg.Add(len(notifications))
			for _, notification := range notifications {
				go func(not SNotification) {
					not.GetSNotification(solver, c, d)
					wg.Done()
				}(notification)
			}
			// run rule cr6
			wg.Add(1)
			go func() {
				solver.cr6.GetSNotification(solver, c, d)
				wg.Done()
			}()
			// apply subset notifications for cr6
			wg.Add(1)
			go func() {
				solver.AllChangesRuleMap.ApplySubsetNotification(solver, c, d)
				wg.Done()
			}()
			wg.Wait()
		case len(solver.pendingRUpdates) != 0:
			// get next r update and apply it concurrently
			n := len(solver.pendingRUpdates)
			next := solver.pendingRUpdates[n-1]
			solver.pendingRUpdates[n-1] = nil
			solver.pendingRUpdates = solver.pendingRUpdates[:n-1]
			// do notifications, again create a waitgroup
			r, c, d := next.R, next.C, next.D
			var wg sync.WaitGroup
			// all notifications waiting for r
			notifications := solver.RRules[r]
			wg.Add(len(notifications))
			for _, notification := range notifications {
				go func(not RNotification) {
					not.GetRNotification(solver, r, c, d)
					wg.Done()
				}(notification)
			}
			// now also inform CR5 (and whatever is there)
			notifications = solver.RRules[uint(NoRole)]
			wg.Add(len(notifications))
			for _, notification := range notifications {
				go func(not RNotification) {
					not.GetRNotification(solver, r, c, d)
					wg.Done()
				}(notification)
			}
			wg.Wait()
		case solver.graphChanged:
			solver.cr6.GetGraphNotification(solver)
			solver.graphChanged = false
		default:
			break L
		}
	}
}

// Full concurrent solver: Run notifications and updates concurrently.
