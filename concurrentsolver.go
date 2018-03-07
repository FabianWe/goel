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
	// ugly duoMutex fix
	solver.duoMutex.Lock()
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
	solver.duoMutex.Unlock()
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

func (solver *ConcurrentNotificationSolver) AddSubsetRule(c, d uint) bool {
	// this is exactly the same as in AllChangesSolver, but exists just to show
	// that we here don't have to worry about concurrency

	res := solver.newSubsetRule(c, d)
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
			// TODO changed the position of graph changed, correct?
			solver.graphChanged = false
			solver.cr6.GetGraphNotification(solver)
		default:
			break L
		}
	}
}

// Full concurrent solver: Run notifications and updates concurrently.

type ConcurrentWorkerPool struct {
	sChan   chan *SUpdate
	rChan   chan *RUpdate
	workers chan bool
	wg      *sync.WaitGroup
}

// TODO document that init must be called
func NewConcurrentWorkerPool() *ConcurrentWorkerPool {
	return &ConcurrentWorkerPool{}
}

func (p *ConcurrentWorkerPool) Init(sSize, rSize, workers int) {
	p.sChan = make(chan *SUpdate, sSize)
	p.rChan = make(chan *RUpdate, rSize)
	p.workers = make(chan bool, workers)
	var wg sync.WaitGroup
	p.wg = &wg
}

func (p *ConcurrentWorkerPool) AddS(update ...*SUpdate) {
	// p.wg.Add(1)
	// go func() {
	// 	p.sChan <- update
	// }()

	p.wg.Add(len(update))
	go func() {
		for _, u := range update {
			p.sChan <- u
		}
	}()
}

func (p *ConcurrentWorkerPool) AddR(update *RUpdate) {
	p.wg.Add(1)
	go func() {
		p.rChan <- update
	}()
}

func (p *ConcurrentWorkerPool) Close() {
	close(p.sChan)
	close(p.rChan)
	// should not be required, but just to be sure
	close(p.workers)
}

func (p *ConcurrentWorkerPool) Wait() {
	p.wg.Wait()
}

// TODO add discussion here involving everything concurrent vs. part of it
// concurrent
func (p *ConcurrentWorkerPool) SWorker(solver *ConcurrentSolver) {
	for update := range p.sChan {
		// first wait for a worker to free
		p.workers <- true
		// now start a go routine that does all updates concurrently
		go func(update *SUpdate) {
			// once we're done we signal that to wg and free the worker
			// defer func() {
			// 	p.wg.Done()
			// 	<-p.workers
			// }()
			c, d := update.C, update.D
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
				// TODO is this correctly protected for concurrent use?
				solver.AllChangesRuleMap.ApplySubsetNotification(solver, c, d)
				wg.Done()
			}()
			<-p.workers
			p.wg.Done()
			wg.Wait()
		}(update)
	}
}

func (p *ConcurrentWorkerPool) RWorker(solver *ConcurrentSolver) {
	for update := range p.rChan {
		// first wait for a worker to free
		p.workers <- true
		// now start a go routine that does all updates concurrently
		go func(update *RUpdate) {
			// once we're done we signal that to wg and free the worker
			defer func() {
				p.wg.Done()
				<-p.workers
			}()
			r, c, d := update.R, update.C, update.D
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
		}(update)
	}
}

type ConcurrentSolver struct {
	*AllChangesSolverState
	*AllChangesRuleMap

	graphChanged      bool
	graphChangedMutex *sync.Mutex

	graph ConceptGraph

	search ExtendedReachabilitySearch

	pool *ConcurrentWorkerPool

	SChanSize, RChanSize, Workers int
}

// Again some code duplication here, well...

func NewConcurrentSolver(graph ConceptGraph, search ExtendedReachabilitySearch) *ConcurrentSolver {
	var graphChangedMutex sync.Mutex
	if search == nil {
		search = BFSToSet
	}
	return &ConcurrentSolver{
		AllChangesSolverState: nil,
		AllChangesRuleMap:     nil,
		graphChanged:          false,
		graphChangedMutex:     &graphChangedMutex,
		graph:                 graph,
		search:                search,
		pool:                  NewConcurrentWorkerPool(),
		SChanSize:             -1,
		RChanSize:             -1,
		Workers:               -1,
	}
}

func (solver *ConcurrentSolver) initPool(tbox *NormalizedTBox) {
	// TODO add some useful defaults here...
	sChanSize, rChanSize, workers := solver.SChanSize, solver.RChanSize, solver.Workers
	if sChanSize < 0 {
		sChanSize = int(tbox.Components.NumBCD() * 4)
	}
	if rChanSize < 0 {
		rChanSize = sChanSize
	}
	if workers < 0 {
		workers = 4
	}
	solver.pool.Init(sChanSize, rChanSize, workers)
}

func (solver *ConcurrentSolver) Init(tbox *NormalizedTBox) {
	solver.graphChanged = false
	// initialize state and rules (concurrently)
	var wg sync.WaitGroup
	wg.Add(3)
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
	go func() {
		solver.initPool(tbox)
		wg.Done()
	}()
	wg.Wait()
}

func (solver *ConcurrentSolver) AddConcept(c, d uint) bool {
	res := solver.AllChangesSolverState.AddConcept(c, d)
	if res {
		update := NewSUpdate(c, d)
		solver.pool.AddS(update)
	}
	return res
}

// TODO experiment
func (solver *ConcurrentSolver) UnionConcepts(c, d uint) bool {
	// we don't want to iterate over each concept twice (once in the set union
	// and once here) so we simply do this by hand... Bit of code duplication
	// but I guess that's okay

	// first we want to avoid some deadlocks (if c == d nothing happens but we
	// can't read / write at the same time)
	if c == d {
		return false
	}
	// ugly duoMutex fix
	solver.duoMutex.Lock()
	solver.sMutex[c].Lock()
	solver.sMutex[d].RLock()
	sc := solver.S[c].M
	sd := solver.S[d].M
	added := false

	vals := make([]*SUpdate, 0, len(sd))

	for v, _ := range sd {
		// add to S(C)
		oldLen := len(sc)
		sc[v] = struct{}{}
		if oldLen != len(sc) {
			// change took place, update
			added = true
			update := NewSUpdate(c, v)
			vals = append(vals, update)
			// solver.pool.AddS(update)
		}
	}
	solver.pool.AddS(vals...)
	solver.sMutex[c].Unlock()
	solver.sMutex[d].RUnlock()
	solver.duoMutex.Unlock()
	return added
}

// func (solver *ConcurrentSolver) UnionConcepts(c, d uint) bool {
// 	// we don't want to iterate over each concept twice (once in the set union
// 	// and once here) so we simply do this by hand... Bit of code duplication
// 	// but I guess that's okay
//
// 	// first we want to avoid some deadlocks (if c == d nothing happens but we
// 	// can't read / write at the same time)
// 	if c == d {
// 		return false
// 	}
// 	// ugly duoMutex fix
// 	solver.duoMutex.Lock()
// 	solver.sMutex[c].Lock()
// 	solver.sMutex[d].RLock()
// 	sc := solver.S[c].M
// 	sd := solver.S[d].M
// 	added := false
//
// 	for v, _ := range sd {
// 		// add to S(C)
// 		oldLen := len(sc)
// 		sc[v] = struct{}{}
// 		if oldLen != len(sc) {
// 			// change took place, update
// 			added = true
// 			update := NewSUpdate(c, v)
// 			solver.pool.AddS(update)
// 		}
// 	}
// 	solver.sMutex[c].Unlock()
// 	solver.sMutex[d].RUnlock()
// 	solver.duoMutex.Unlock()
// 	return added
// }

func (solver *ConcurrentSolver) AddRole(r, c, d uint) bool {
	res := solver.AllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue pending update
		update := NewRUpdate(r, c, d)
		// add update
		solver.pool.AddR(update)
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

// TODO right?
func (solver *ConcurrentSolver) AddSubsetRule(c, d uint) bool {
	return solver.newSubsetRule(c, d)
}

func (solver *ConcurrentSolver) Solve(tbox *NormalizedTBox) {
	// TODO call init here, made this easier for testing during debuging.
	// now run the listeners and apply the initial updates
	addInitial := func() {
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
	}

	// TODO moved up from below
	// solver.initPool(tbox)
	go solver.pool.RWorker(solver)
	go solver.pool.SWorker(solver)

	solver.runAndWait(tbox, addInitial)

	// after the initial stuff has been added: while the graph has changed
	// run the graph update and apply all the rules

	for solver.graphChanged {
		// apply rule and wait until all changes have happened, then if the graph
		// changed again repeat the process.
		// all other updates must have already taken place
		solver.graphChanged = false
		f := func() {
			solver.cr6.GetGraphNotification(solver)
		}
		solver.runAndWait(tbox, f)
	}

	// TODO moved up from below
	solver.pool.Close()
}

// TODO it's not okay to run f and the workers concurrently, that is because
// CR6 locking is not very efficient and should be optimised because it leads
// to deadlocks
// see commented out version below.
// but still there is much running concurrently, so I'm okay with that.
// seems that the problem really was the union method, since we fixed that
// (in an ugly way though) the version below seems to work just fine.
// but think this through again... why exactly did it deadlock before and
// why not any more? and why did the union deadlock never occur here?
// Another problem: imagine that all workers are locked (and this means for
// example they want to change the graph but they can't). Can they, at the same
// time, all lock one of the S(C) for which we must apply a union?
// Think this through again! If deadlocks appear use this function just to be
// sure.
// I think not, the following happens: The GetGraphNotification function will
// readlock the mutex, so the only reason one of the workers might hang (or all
// of the for that matter) is because they want to right to the graph, which
// is of course not possible.but these methods don't keep a lock on any S(C),
// so it's perfectly safe for the graph to do the unions.
// func (solver *ConcurrentSolver) runAndWait(tbox *NormalizedTBox, f func()) {
// 	// initialize pool again
// 	solver.initPool(tbox)
// 	// first let f do everything it needs
// 	f()
// 	// now start the listeners
// 	go solver.pool.RWorker(solver)
// 	go solver.pool.SWorker(solver)
// 	// wait until everything is done
// 	solver.pool.Wait()
// 	solver.pool.Close()
// }

// TODO do we really have to close and re-init the the channel again?
// also do we have to start the listeners again and again?
// I don't think so, needs testing though
func (solver *ConcurrentSolver) runAndWait(tbox *NormalizedTBox, f func()) {
	// initialize pool again

	// TODO has been moved up
	// solver.initPool(tbox)

	// we want to run a listener for s and r, also we would like to run f
	// concurrently and then wait
	// but we have to wait until f has been applied

	// TODO has been moved up, hope this is still correct
	// go solver.pool.RWorker(solver)
	// go solver.pool.SWorker(solver)

	// so now we run f concurrently, report back to a done channel once it's
	// finished and then we have to wait until all updates are done (i.e. wait
	// for the pool)
	done := make(chan bool, 1)
	go func() {
		f()
		done <- true
	}()
	// now wait until f is finished
	<-done
	// now wait until the pool is done
	solver.pool.Wait()

	// TODO has been moved up as well
	// solver.pool.Close()
}
