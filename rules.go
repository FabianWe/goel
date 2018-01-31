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
// state information. That is update the mappings S(C) and R(r) and check if
// certain elements are present in it.
// For R(r) it is sometimes required to iterate over each element in R(r),
// therefor exists methods that take a channel as an input, write each element
// to that channel and then close the channel. Thus methods that require to
// iterate over each R(r) start a go routine with a channel and iterate the
// channel until it is closed.
// A basic implementation is given in SolverState, other state handlers or even
// solvers can simply delegate certain methods to this basic implementation.
// That is they can add additional logic (such as triggering certain rules)
// and let the SolverState handle everything else. Of course it is also
// possible to write a completely new handler.
//
// State handlers must be safe to use concurrently from multiple go routines.
// That is also true for methods that iterate over objects: No write operations
// may happen during that time.
// TODO update documentation regarding graph stuff
type StateHandler interface {
	// ContainsConcept checks whether D ∈ S(C).
	ContainsConcept(c, d uint) bool

	// AddConcept adds D to S(C) and returns true if the update changed S(C).
	AddConcept(c, d uint) bool

	// UnionConcepts adds all elements from S(D) to S(C), thus performs the update
	// S(C) = S(C) ∪ S(D). Returns true if some elements were added to S(C).
	UnionConcepts(c, d uint) bool

	// ContainsRole checks whether (C, D) ∈ R(r).
	ContainsRole(r, c, d uint) bool

	// AddRole adds (C, D) to R(r). It must also update the graph.
	// The first boolean signals if an update in R(r) occurred, the second
	// if an update in the graph occurred.
	// TODO This is not really nice, we probably need something else later...
	// and is ignored in the rules now.
	AddRole(r, c, d uint) bool

	// RoleMapping returns all pairs (C, D) in R(r) for a given C.
	RoleMapping(r, c uint, ch chan<- uint)

	// ReverseRoleMapping returns all pairs (C, D) in R(r) for a given D.
	ReverseRoleMapping(r, d uint, ch chan<- uint)

	// GetComponents returns the number of all objects, s.t. we can use it when
	// needed.
	GetComponents() *ELBaseComponents
}

// SolverState is an implementation of StateHandler, for more details see there.
// It protects each S(C) and R(r) with a RWMutex.
type SolverState struct {
	components *ELBaseComponents

	S []*BCSet
	R []*Relation

	sMutex []*sync.RWMutex
	rMutex []*sync.RWMutex
}

// NewSolverState returns a new solver state given the base components,
// thus it initializes S and R and the mutexes used to control r/w access.
func NewSolverState(c *ELBaseComponents) *SolverState {
	res := SolverState{
		components: c,
		S:          nil,
		R:          nil,
		sMutex:     nil,
		rMutex:     nil,
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
	relationChanged := state.R[r].Add(c, d)
	state.rMutex[r].Unlock()
	return relationChanged
}

func (state *SolverState) RoleMapping(r, c uint, ch chan<- uint) {
	state.rMutex[r].RLock()
	m := state.R[r].mapping[c]
	for d, _ := range m {
		ch <- d
	}
	close(ch)
	state.rMutex[r].RUnlock()
}

func (state *SolverState) ReverseRoleMapping(r, d uint, ch chan<- uint) {
	state.rMutex[r].RLock()
	m := state.R[r].reverseMapping[d]
	for c, _ := range m {
		ch <- c
	}
	close(ch)
	state.rMutex[r].RUnlock()
}

func (state *SolverState) GetComponents() *ELBaseComponents {
	return state.components
}

type SNotification interface {
	// Information, that C' was added to S(C)
	GetSNotification(state StateHandler, c, cPrime uint) bool
}

type RNotification interface {
	// Information, that (C, D) was added to R(r)
	GetRNotification(state StateHandler, r, c, d uint) bool
}

// TODO check for deadlocks... don't read from something and then lock it.
// TODO probable problem: Iterating over R(r) and adding to R(r) in the same
// loop, I think this can happen... we have to check that!
// The easiest way is probably to simply don't try to add anything to R(r)
// while iterating over it since it will never change anything.

// Now follow the rules, information of how to understand the rules are given
// in their documentation.

// CR1 implements the rule CR1: for C' ⊑ D:
// If C' ∈ S(C) then add D to S(C).
//
// This rule implements an SNotification and as a result must simply add
// D to S(C). We only need to remember D (assuming the rule was correctly
// added). That is the value of the uint.
//
// The intended use is that such a notification is created for each C' ⊑ D
// and then listens to all S(C) until C' gets added.
type CR1 uint

func NewCR1(d uint) CR1 {
	return CR1(d)
}

func (n CR1) GetSNotification(state StateHandler, c, cPrime uint) bool {
	// we have to add D (from the rule) to S(C)
	return state.AddConcept(c, uint(n))
}

// CR2 implements the rule CR2: for C1 ⊓ C2 ⊑ D:
// If C1, C2 ∈ S(C) add D to S(C).
//
// This rule implements an SNotification that waits for updates on any C
// with either C1 or C2. If triggered it checks if both values are present in
// S(C) and as a result adds D to S(C).
//
// The intended use is that such a notification is created for each C1 ⊓ C2 ⊑ D
// and then this instance listens on both C1 and C2 for all S(C).
type CR2 struct {
	C1, C2, D uint
}

func NewCR2(c1, c2, d uint) *CR2 {
	return &CR2{c1, c2, d}
}

func (n *CR2) GetSNotification(state StateHandler, c, cPrime uint) bool {
	// do a lookup for the other value, if both are found try to apply rule
	otherFound := false
	switch cPrime {
	case n.C1:
		otherFound = state.ContainsConcept(c, n.C2)
	case n.C2:
		otherFound = state.ContainsConcept(c, n.C1)
	default:
		// TODO remove once tested
		log.Printf("Wrong handler for rule CR2: Got update for %d, but only listening to %d and %d",
			cPrime, n.C1, n.C2)
		return false
	}
	return otherFound && state.AddConcept(c, n.D)
}

// CR3 implements the rule CR3: for C' ⊑ ∃r.D:
// If C' ∈ S(C) add (C, D) to R(r).
//
// This rule implements an SNotificationthat waits for updates on any C with
// C'. When triggered it directly performs the update.
//
// The intended use is that such a notification is created for each C' ⊑ ∃r.D
// and then waits for updates on all S(C) with C'.
type CR3 struct {
	R, D uint
}

func NewCR3(r, d uint) *CR3 {
	return &CR3{r, d}
}

func (n *CR3) GetSNotification(state StateHandler, c, cPrime uint) bool {
	return state.AddRole(n.R, c, n.D)
}

// CR4 implements the rule CR4: for ∃r.D' ⊑ E:
// If (C, D) ∈ R(r) and D' ∈ S(D) then add E to S(C).
// It implements both SNotification and RNotification.
//
// On an update on R(r) with (C, D) it checks if D' ∈ S(D). If yes the update
// is applied.
//
// On an update on S(D) with D' it checks all pairs (C, D) ∈ R(r) and applies
// the update for these pairs.
//
// The intended use is that such a notification is created for each ∃r.D' ⊑ E
// and then waits for updates on all S(D) with D' and for any update on R(r)
// (for that particular r).
type CR4 struct {
	R, DPrime, E uint
}

func NewCR4(r, dPrime, e uint) *CR4 {
	return &CR4{r, dPrime, e}
}

func (n *CR4) GetRNotification(state StateHandler, r, c, d uint) bool {
	// TODO maybe add some debugging messages, but I'm too lazy for that now
	// check if dprime is in S(D) and then try to add E to S(C)
	return state.ContainsConcept(d, n.DPrime) && state.AddConcept(c, n.E)
}

func (n *CR4) GetSNotification(state StateHandler, d, dPrime uint) bool {
	// TODO maybe again some debug messages...
	// iterate over each (C, D) ∈ R(r)
	ch := make(chan uint, 1)
	go state.ReverseRoleMapping(n.R, d, ch)
	// iterate over each c
	// TODO union could be nicer here in order to avoid too many locks... test it
	res := false
	for c := range ch {
		res = state.AddConcept(c, n.E) || res
	}
	return res
}

// CR5 implements the rule CR5: If (C, D) ∈ R(r) and ⊥ ∈ S(D) then add ∈ to
// S(C).
//
// It implements both SNotification and RNotification.
//
// On an update on R(r) with (C, D) it checks if ⊥ ∈ S(D) and if yes applies
// the rule.
//
// On an update on S(D) with ⊥ it iterates over all pairs (C, D) in R(r) for all
// r and applies the rule. That is a rather cumbersome process but it can't be
// helped.
//
// This notification should be created once and then listen on all r and all D
// (for ⊥).
type CR5 struct{}

func NewCR5() *CR5 {
	return &CR5{}
}

func (n *CR5) GetRNotification(state StateHandler, r, c, d uint) bool {
	// check if ⊥ ∈ S(D) and then try to add ⊥ to c
	return state.ContainsConcept(d, 0) && state.AddConcept(c, 0)
}

func (n *CR5) GetSNotification(state StateHandler, d, bot uint) bool {
	// TODO maybe we could add some concurrency here...
	if bot != 0 {
		log.Printf("Error in rule CR5: Expected bottom concept, but got %d", bot)
		return false
	}
	res := false
	numR := state.GetComponents().Roles
	var r uint
	for ; r < numR; r++ {
		ch := make(chan uint, 1)
		go state.ReverseRoleMapping(r, d, ch)
		for c := range ch {
			res = state.AddConcept(c, 0) || res
		}
	}
	return res
}

// TODO CR6

// CR10 implements the rule CR10: for r ⊑ s:
// If (C, D) ∈ R(r) then add (C, D) to R(s).
//
// This rule implements RNotification and as a result simply adds (C, D) to
// R(S). We only need to remember s (assuming the rule was correctly added).
// That is the value of the uint.
//
// The intended use is that such a notification is created for each r ⊑ s
// and then listens to changes on R(r) (for that specific r).
type CR10 uint

func NewCR10(s uint) CR10 {
	return CR10(s)
}

func (n CR10) GetRNotification(state StateHandler, r, c, d uint) bool {
	return state.AddRole(uint(n), c, d)
}

// CR11 implements the rule CR11: for r1 o r2 ⊑ r3:
// If (C, D) ∈ R(r1) and (D, E) ∈ R(r2) then add (C, E) to R(r3).
//
// This rule implements RNotification and waits on changes for both r1 and r2.
// Updates require iterating over R(r1) / R(r2).
//
// The intended use is that such a notification is created for each r1 o r2 ⊑ r3
// and then listens to changes on R(r1) and R(r2).
type CR11 struct {
	R1, R2, R3 uint
}

func NewCR11(r1, r2, r3 uint) *CR11 {
	return &CR11{r1, r2, r3}
}

func (n *CR11) GetRNotification(state StateHandler, r, c, d uint) bool {
	switch r {
	case n.R1:
		ch := make(chan uint, 1)
		go state.RoleMapping(n.R2, d, ch)
		result := false
		for e := range ch {
			result = state.AddRole(n.R3, c, e) || result
		}
		return result
	case n.R2:
		// first some renaming to keep it more readable...
		// in this case the names in the rule are (D, E) for R(r2)
		e := d
		d = c
		ch := make(chan uint, 1)
		go state.ReverseRoleMapping(n.R1, d, ch)
		result := false
		for c := range ch {
			result = state.AddRole(n.R3, c, e) || result
		}
		return result
	default:
		// TODO remove once tested
		log.Printf("Invalid notification for rule CR11: Only waiting for changes on %d and %d, got %d",
			n.R1, n.R2, r)
		return false
	}
}

// RuleMap is used to store all rules in a way in which we can easily determin
// which rules are to be notified about a certain change.
//
// There are two types of notifications: SNotification which handles updates
// of the form "C' was added to S(C)" and RNotification which handles updates
// of the form "(C, D) was added to R(r)".
//
// SNotifications (or better to say rules they represent) are always of the
// form that they listen for the change made to any C and wait until a certain
// value is added to that C.
//
// For example consider rule CR1: If C' ∈ S(C), C' ⊑ D then S(C) = S(C) ∪ {D}
// That means: We only wait for an update with the value C' (that's the only
// thing that can trigger this rule). Thus when we add C' to some S(C) we
// lookup which notifications are interested in this update (CR1 being one
// of them) and inform them about this update.
// We implement this by a map that maps C' → list of notifications.
// This means: When C' is added to some C inform all notifications in map[C'].
//
// Rules waiting for some R(r) are organized a bit diffent: They don't want to
// be informed about a certain (C, D) being added, but want to be informed about
// all (C, D) that are added.
//
// Most of the rules want to listen only on a certain r, for example rule
// CR4 says that if we have ∃r.D' ⊑ E we have to listen to all elements added
// to R(r) for that specific r.
// Rule CR5 on the other hand waits on updates on all roles.
// Thus we have a map that maps r → list of notifications. This list holds
// all notifications that are interested in r. It also contains an entry
// for NoRole (which is used to describe an id that is not really a role)
// and stores all notifcations interested in updates on all R(r) (should only
// be CR5).
//
// A RuleMap is initialized with a given normalized TBox and creates all
// notifications and adds them. Thus before usage the Init method must be
// called with that TBox.
// If other rules are required and should be added it should be noted that
// it is not safe for concurrent writing acces.
//
// If it is really required see the worker methods (should not be needed if
// you just want to initialize it with a given TBox).
type RuleMap struct {
	SRules map[uint][]SNotification
	RRules map[uint][]RNotification
}

func NewRuleMap() *RuleMap {
	return &RuleMap{make(map[uint][]SNotification), make(map[uint][]RNotification)}
}

type AddSNotification struct {
	value        uint
	notification SNotification
}

func NewAddSNotification(value uint, notification SNotification) AddSNotification {
	return AddSNotification{value, notification}
}

type AddRNotification struct {
	role         uint
	notification RNotification
}

func NewAddRNotification(role uint, notification RNotification) AddRNotification {
	return AddRNotification{role, notification}
}

// AddSWorker is a little helper method that is used to concurrently add
// new entries to SRules.
// Start a gourotine for that message, write all notifications to the channel
// ch, close the channel once you're done and wait on the done channel until
// all updates certainly happened.
func (rm *RuleMap) AddSWorker(ch <-chan AddSNotification, done chan<- bool) {
	for add := range ch {
		rm.SRules[add.value] = append(rm.SRules[add.value], add.notification)
	}
	done <- true
}

// AddRWorker works similar as AddSWorker, only for RNotifications.
func (rm *RuleMap) AddRWorker(ch <-chan AddRNotification, done chan<- bool) {
	for add := range ch {
		rm.RRules[add.role] = append(rm.RRules[add.role], add.notification)
	}
	done <- true
}

// TODO add missing rule(s?)! CR5 / CR6!
func (rm *RuleMap) Init(tbox *NormalizedTBox) {
	components := tbox.Components
	// we start both workers s.t. we can concurrently add new notifications,
	// then we build the rules
	sChan := make(chan AddSNotification, 1)
	rChan := make(chan AddRNotification, 1)
	done := make(chan bool)

	// start go routines
	go rm.AddSWorker(sChan, done)
	go rm.AddRWorker(rChan, done)

	var wg sync.WaitGroup
	wg.Add(5)

	// start a goroutine for all initialisation steps

	// Normalized CIs
	go func() {
		defer wg.Done()
		for _, ci := range tbox.CIs {
			if ci.C2 == nil {
				// rule CR1
				cPrime := ci.C1.NormalizedID(components)
				d := ci.D.NormalizedID(components)
				// add rule
				cr1 := NewCR1(d)
				add := NewAddSNotification(cPrime, cr1)
				sChan <- add
			} else {
				c1 := ci.C1.NormalizedID(components)
				c2 := ci.C2.NormalizedID(components)
				d := ci.D.NormalizedID(components)
				// create rule
				cr2 := NewCR2(c1, c2, d)
				// add for both c1 and c2
				add1 := NewAddSNotification(c1, cr2)
				add2 := NewAddSNotification(c2, cr2)
				sChan <- add1
				sChan <- add2
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
			cr3 := NewCR3(r, d)
			add := NewAddSNotification(cPrime, cr3)
			sChan <- add
		}
	}()

	// NormalizedCILeftEx
	go func() {
		defer wg.Done()
		for _, ex := range tbox.CILeft {
			r := uint(ex.R)
			dPrime := ex.C1.NormalizedID(components)
			e := ex.D.NormalizedID(components)
			cr4 := NewCR4(r, dPrime, e)
			adds := NewAddSNotification(dPrime, cr4)
			addr := NewAddRNotification(r, cr4)
			sChan <- adds
			rChan <- addr
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
				cr10 := NewCR10(s)
				add := NewAddRNotification(r, cr10)
				rChan <- add
			} else {
				// CR11
				r1 := uint(ri.R1)
				r2 := uint(ri.R2)
				r3 := uint(ri.S)
				cr11 := NewCR11(r1, r2, r3)
				first := NewAddRNotification(r1, cr11)
				second := NewAddRNotification(r2, cr11)
				rChan <- first
				rChan <- second
			}
		}
	}()

	// add CR5
	go func() {
		defer wg.Done()
		cr5 := NewCR5()
		// add listener for ⊥
		cr5s := NewAddSNotification(0, cr5)
		cr5r := NewAddRNotification(uint(NoRole), cr5)
		sChan <- cr5s
		rChan <- cr5r
	}()

	// wait until everything has been added to the channels
	wg.Wait()
	// so both workers can stop
	close(sChan)
	close(rChan)
	// wait until all elements have been added by the workers
	<-done
	<-done
}
