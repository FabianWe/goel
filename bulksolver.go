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

	"github.com/FabianWe/goel/domains"
)

// BulkWorker is a helper class for the BulkSolver.
// It can be used as a solver state that adds consequences to an internal queue
// and changes the global mappings S and R (given from the BulkSolver).
// S and R are updated / queried through the AllChangesSolverState of the
// linked BulkSolver directly, i.e. it does not call for example AddConcept
// on the bulk solver but on the solver state of that solver. That way
// the bulk solver can process all consequences from the queues.
type BulkWorker struct {
	*AllChangesSolverState
	*AllChangesRuleMap
	bulkSolver *BulkSolver

	ComputedR []*RUpdate
	ComputedS []*SUpdate
}

func NewBulkWokrer(solver *BulkSolver) *BulkWorker {
	return &BulkWorker{
		AllChangesSolverState: solver.AllChangesSolverState,
		AllChangesRuleMap:     solver.AllChangesRuleMap,
		bulkSolver:            solver,
		ComputedR:             nil,
		ComputedS:             nil,
	}
}

func (worker *BulkWorker) AddConcept(c, d uint) bool {
	res := worker.AllChangesSolverState.AddConcept(c, d)
	if res {
		worker.bulkSolver.wg.Add(1)
		update := NewSUpdate(c, d)
		worker.ComputedS = append(worker.ComputedS, update)
	}
	return res
}

func (worker *BulkWorker) UnionConcepts(c, d uint) bool {
	// we don't want to iterate over each concept twice (once in the set union
	// and once here) so we simply do this by hand... Bit of code duplication
	// but I guess that's okay

	// first we want to avoid some deadlocks (if c == d nothing happens but we
	// can't read / write at the same time)
	if c == d {
		return false
	}
	// ugly duoMutex fix
	worker.duoMutex.Lock()
	worker.sMutex[c].Lock()
	worker.sMutex[d].RLock()
	sc := worker.S[c].M
	sd := worker.S[d].M
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
		}
	}

	worker.sMutex[c].Unlock()
	worker.sMutex[d].RUnlock()
	worker.duoMutex.Unlock()
	worker.bulkSolver.wg.Add(len(vals))
	worker.ComputedS = append(worker.ComputedS, vals...)
	return added
}

func (worker *BulkWorker) AddRole(r, c, d uint) bool {
	res := worker.bulkSolver.AllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue pending update
		update := NewRUpdate(r, c, d)
		worker.bulkSolver.wg.Add(1)
		// add update
		worker.ComputedR = append(worker.ComputedR, update)
		// update graph
		worker.bulkSolver.graphMutex.Lock()
		graphUpdate := worker.bulkSolver.Graph.AddEdge(c, d)
		worker.bulkSolver.graphMutex.Unlock()
		// if update changed something notify about the update
		if graphUpdate {
			worker.bulkSolver.graphChangedMutex.Lock()
			worker.bulkSolver.graphChanged = true
			worker.bulkSolver.graphChangedMutex.Unlock()
		}
	}
	return res
}

// TODO right?
func (worker *BulkWorker) AddSubsetRule(c, d uint) bool {
	return worker.bulkSolver.newSubsetRule(c, d)
}

// Run runs all updates and adds the derived consequences to the internal
// queues.
func (worker *BulkWorker) Run(sUpdates []*SUpdate, rUpdates []*RUpdate) {
	for _, update := range sUpdates {
		c, d := update.C, update.D
		// first lookup all rules that are interested in an update
		// on S(D)
		notifications := worker.bulkSolver.SRules[d]
		// now iterate over each notification and apply it
		for _, notification := range notifications {
			notification.GetSNotification(worker, c, d)
		}
		// now also do a notification for CR6
		worker.cr6.GetSNotification(worker, c, d)
		// apply subset notifications for cr6
		worker.AllChangesRuleMap.ApplySubsetNotification(worker, c, d)
		// apply notification for CR7/CR8
		worker.cr7A8.GetSNotification(worker, c, d)
	}
	for _, update := range rUpdates {
		// do notifications for the update
		r, c, d := update.R, update.C, update.D
		// first all notifications waiting for r
		notifications := worker.RRules[r]
		for _, notification := range notifications {
			notification.GetRNotification(worker, r, c, d)
		}
		// now inform CR5 (or however else is waiting on an update on all roles)
		notifications = worker.RRules[uint(NoRole)]
		for _, notification := range notifications {
			notification.GetRNotification(worker, r, c, d)
		}
	}
}

// BulkSolver is a solver that runs concurrently but tries to avoid that
// consequences of a single update are performed concurrently, instead updates
// are accumulated and then performed in bulks.
//
// After creating a BulkSolver via NewBulkSolver the following settings can
// be changed before solving:
// Workers: The number of workers that are allowed to run concurrently.
// A worker will process a set of updates concurrently, i.e. use BulkWorker
// to compute all consequences.
//
// K is an option that stores how many updates at most are processed by a single
// worker. Setting K to a lower value usually means that more updates will be
// processed concurrently (instead of running them all at once).
//
// Internally it works by a method that waits for updates, computes the bulks
// and runs them concurrently. This method may get deactivated and then
// activated again.
type BulkSolver struct {
	*AllChangesSolverState
	*AllChangesRuleMap

	graphChanged      bool
	graphChangedMutex *sync.Mutex

	graph ConceptGraph

	search ExtendedReachabilitySearch

	wg *sync.WaitGroup

	// K is the number of updates that will be run by a concurrent worker
	// (maximal K things, maybe less). This is a way to let the solver know
	// how many updates will be processed without concurrency.
	// If set to a number ≤ 0 everything that is currently at the queue gets
	// processed in parallel.
	// TODO add a nice way to find a good k automatically?
	K int

	// stuff for blocking the system, that is pending updates and the boolean flag
	// if a listener is active and the
	block           *sync.Mutex
	pendingSUpdates []*SUpdate
	pendingRUpdates []*RUpdate
	isActivated     bool

	// stuff for the workers
	Workers    int
	WorkerPool chan bool
}

func NewBulkSolver(graph ConceptGraph, search ExtendedReachabilitySearch) *BulkSolver {
	var graphChangedMutex sync.Mutex
	var wg sync.WaitGroup
	var block sync.Mutex
	if search == nil {
		search = BFSToSet
	}
	return &BulkSolver{
		AllChangesSolverState: nil,
		AllChangesRuleMap:     nil,
		graphChanged:          false,
		graphChangedMutex:     &graphChangedMutex,
		graph:                 graph,
		search:                search,
		wg:                    &wg,
		K:                     -1,
		block:                 &block,
		pendingSUpdates:       nil,
		pendingRUpdates:       nil,
		isActivated:           false,
		Workers:               -1,
		WorkerPool:            nil,
	}
}

func (solver *BulkSolver) AddConcept(c, d uint) bool {
	res := solver.AllChangesSolverState.AddConcept(c, d)
	if res {
		solver.wg.Add(1)
		update := NewSUpdate(c, d)
		// add to pending queue and activate
		solver.block.Lock()
		solver.pendingSUpdates = append(solver.pendingSUpdates, update)
		solver.activate()
		// unlock
		solver.block.Unlock()
	}
	return res
}

// TODO is this even required? will this ever be called? I guess so...

func (solver *BulkSolver) UnionConcepts(c, d uint) bool {
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
	solver.sMutex[c].Unlock()
	solver.sMutex[d].RUnlock()
	solver.duoMutex.Unlock()
	// add to pending queue and activate
	solver.wg.Add(len(vals))
	solver.block.Lock()
	solver.pendingSUpdates = append(solver.pendingSUpdates, vals...)
	solver.activate()
	// unlock
	solver.block.Unlock()
	return added
}

func (solver *BulkSolver) AddRole(r, c, d uint) bool {
	res := solver.AllChangesSolverState.AddRole(r, c, d)
	if res {
		// update graph as well and issue pending update
		update := NewRUpdate(r, c, d)
		solver.wg.Add(1)
		solver.block.Lock()
		solver.pendingRUpdates = append(solver.pendingRUpdates, update)
		solver.activate()
		// unlock
		solver.block.Unlock()
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

func (solver *BulkSolver) AddSubsetRule(c, d uint) bool {
	return solver.newSubsetRule(c, d)
}

func (solver *BulkSolver) Init(tbox *NormalizedTBox, domains *domains.CDManager) {
	solver.graphChanged = false
	// initialize state and rules (concurrently)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		solver.AllChangesSolverState = NewAllChangesSolverState(tbox.Components,
			domains, solver.graph, solver.search)
		wg.Done()
	}()
	go func() {
		solver.AllChangesRuleMap = NewAllChangesRuleMap()
		solver.AllChangesRuleMap.Init(tbox)
		wg.Done()
	}()
	workers := solver.Workers
	if workers <= 0 {
		// TODO add some sane defaults?
		workers = 25
	}
	solver.WorkerPool = make(chan bool, workers)
	wg.Wait()
}

// activate requires that solver.block is locked!
func (solver *BulkSolver) activate() {
	if solver.isActivated {
		return
	} else {
		go solver.listener()
		solver.isActivated = true
	}
}

func (solver *BulkSolver) listener() {
L:
	for {
		// get a worker
		solver.WorkerPool <- true
		// lock state
		solver.block.Lock()
		n := len(solver.pendingSUpdates)
		m := len(solver.pendingRUpdates)
		if (n + m) > 0 {
			// start a worker
			var sUpdates []*SUpdate
			var rUpdates []*RUpdate
			if solver.K <= 0 {
				sUpdates = make([]*SUpdate, len(solver.pendingSUpdates))
				rUpdates = make([]*RUpdate, len(solver.pendingRUpdates))
				copy(sUpdates, solver.pendingSUpdates)
				copy(rUpdates, solver.pendingRUpdates)
				solver.pendingSUpdates = nil
				solver.pendingRUpdates = nil
			} else {
				numS := IntMin(solver.K, len(solver.pendingSUpdates))
				sUpdates = make([]*SUpdate, numS)
				copy(sUpdates, solver.pendingSUpdates[:numS])
				solver.pendingSUpdates = solver.pendingSUpdates[numS:]
				// if there are still updates we can execute (not reached k yet)
				// also r updates
				if numS < solver.K {
					// now find out how much we can run
					numR := IntMin(solver.K-numS, m)
					rUpdates = make([]*RUpdate, numR)
					copy(rUpdates, solver.pendingRUpdates[:numR])
					solver.pendingRUpdates = solver.pendingRUpdates[numR:]
				}
			}
			// run worker concurrently
			// fmt.Println("Running with", len(sUpdates)+len(rUpdates))
			go func(sUpdates []*SUpdate, rUpdates []*RUpdate) {
				worker := NewBulkWokrer(solver)
				worker.Run(sUpdates, rUpdates)
				// fmt.Println("Done running")
				n := len(sUpdates)
				m := len(rUpdates)
				solver.wg.Add(-(n + m))
				solver.block.Lock()
				solver.pendingSUpdates = append(solver.pendingSUpdates, worker.ComputedS...)
				solver.pendingRUpdates = append(solver.pendingRUpdates, worker.ComputedR...)
				// TODO is this even required? hmm... doesn't hurt in any case
				if (len(worker.ComputedS) + len(worker.ComputedR)) > 0 {
					solver.activate()
				}
				solver.block.Unlock()
				// free worker
				<-solver.WorkerPool
			}(sUpdates, rUpdates)
			solver.block.Unlock()
		} else {
			// release worker, no work to be done
			solver.block.Unlock()
			<-solver.WorkerPool
			break L
		}
	}
	// once we're here there's no work to be done, but new things might have
	// been added concurrently, so check this now
	solver.block.Lock()
	solver.isActivated = false
	n := len(solver.pendingSUpdates)
	m := len(solver.pendingRUpdates)
	if (n + m) > 0 {
		solver.activate()
	}
	solver.block.Unlock()
}

func (solver *BulkSolver) Solve(tbox *NormalizedTBox) {
	// TODO call init here, made this easier for testing during debuging.
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
	solver.runAndWait(addInitial)
	for solver.graphChanged {
		// apply rule and wait until all changes have happened, then if the graph
		// changed again repeat the process.
		// all other updates must have already taken place
		solver.graphChanged = false
		f := func() {
			solver.cr6.GetGraphNotification(solver)
		}
		solver.runAndWait(f)
	}
}

func (solver *BulkSolver) runAndWait(f func()) {
	done := make(chan bool, 1)
	go func() {
		f()
		done <- true
	}()
	// wait until f is finished
	<-done
	// wait until all workers are done
	solver.wg.Wait()
}
