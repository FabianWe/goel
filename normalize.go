// The MIT License (MIT)
//
// Copyright (c) 2016, 2017 Fabian Wenzelmann
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
	"log"
	"reflect"
	"sync"
)

// PhaseOneResult is a helper type that stores certain information about
// normalization in phase 1.
//
// The idea is that each formula that gets normalized uses its own result
// and stores all information inside.
//
// It is not save for concurrent use.
//
// The following information is stored:
//
// Distributor is used to create new names, so it should be the same in
// all objects of this type when normalizing a set of formulae.
// Waiting is a "queue" that stores all elements that need further processing,
// that means it may be that additional rules can be applied in phase 1.
// So when replacing a formula with other formulae just add the new formulae
// to Waiting.
// IntermediateRes is the set of all formulae that must not be processed in
// phase 1 but must be used in phase 2.
// The other intermediate results are used for formulae already in normal form:
// We don't have to check them again in phase 2 so we can already add them here.
type PhaseOneResult struct {
	Distributor     *IntDistributor
	Waiting         []*GCIConstraint
	IntermediateRes []*GCIConstraint
	// for CIs already in normal form we can just add them already for phase 2
	IntermediateCIs     []*NormalizedCI
	IntermediateCILeft  []*NormalizedCILeftEx
	IntermediateCIRight []*NormalizedCIRightEx
}

func NewPhaseOneResult(distributor *IntDistributor) *PhaseOneResult {
	return &PhaseOneResult{Distributor: distributor}
}

// Union merges two results.
// That is it builds the union of all intermediate results.
// This is useful when combining results from different normalizations.
// The combined results are stored in res.
func (res *PhaseOneResult) Union(other *PhaseOneResult) {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		res.IntermediateRes = append(res.IntermediateRes, other.IntermediateRes...)
	}()

	go func() {
		defer wg.Done()
		res.IntermediateCIs = append(res.IntermediateCIs, other.IntermediateCIs...)
	}()

	go func() {
		defer wg.Done()
		res.IntermediateCILeft = append(res.IntermediateCILeft, other.IntermediateCILeft...)
	}()

	go func() {
		defer wg.Done()
		res.IntermediateCIRight = append(res.IntermediateCIRight, other.IntermediateCIRight...)
	}()

	wg.Wait()
}

// PhaseTwoResult stores information about normalization in phase 2.
// The idea is the same as for PhaseOneResult, so see documentation there.
type PhaseTwoResult struct {
	Distributor         *IntDistributor
	Waiting             []*GCIConstraint
	IntermediateCIs     []*NormalizedCI
	IntermediateCILeft  []*NormalizedCILeftEx
	IntermediateCIRight []*NormalizedCIRightEx
}

func NewPhaseTwoResult(distributor *IntDistributor) *PhaseTwoResult {
	return &PhaseTwoResult{Distributor: distributor}
}

func (res *PhaseTwoResult) Union(other *PhaseTwoResult) {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.IntermediateCIs = append(res.IntermediateCIs, other.IntermediateCIs...)
	}()

	go func() {
		defer wg.Done()
		res.IntermediateCILeft = append(res.IntermediateCILeft, other.IntermediateCILeft...)
	}()

	go func() {
		defer wg.Done()
		res.IntermediateCIRight = append(res.IntermediateCIRight, other.IntermediateCIRight...)
	}()

	wg.Wait()
}

// NormalizeStepPhaseOne performs one step of the normalization in phase 1.
// It tests what form the gci has and adds new formulae to the right slice
// of the intermediate result. That is it appends formulae to waiting
// or one of the intermediate results.
//
// Phase one means to apply the rules NF2 - NF4 (NF1 is applied somewhere else).
func NormalizeStepPhaseOne(gci *GCIConstraint, intermediate *PhaseOneResult) {
	// do a case distinction on lhs. Check if gci is already in normal form, if
	// a rule can be applied in phase 1 or if no rule can be applied in phase 1
	switch lhs := gci.C.(type) {
	default:
		log.Printf("Unexepcted type in TBox normalization phase 1: %v\n", reflect.TypeOf(lhs))
	case NamedConcept, NominalConcept, ConcreteDomainExtension, TopConcept:
		// the lhs is in BCD, check if rhs is in BCD or false
		switch rhs := gci.D.(type) {
		case NamedConcept, NominalConcept, ConcreteDomainExtension, TopConcept, BottomConcept:
			// lhs is in BCD and rhs is in BCD or false, so it is already in
			// normal form
			normalized := NewNormalizedCI(lhs, nil, rhs)
			intermediate.IntermediateCIs = append(intermediate.IntermediateCIs, normalized)
		case ExistentialConcept:
			// check if the rhs is of the form ∃r.C2 where C2 is in BCD
			if rhs.C.IsInBCD() {
				// in normal form, add it
				normalized := NewNormalizedCIRightEx(lhs, rhs.R, rhs.C)
				intermediate.IntermediateCIRight = append(intermediate.IntermediateCIRight, normalized)
			} else {
				// not in normal form
				intermediate.IntermediateRes = append(intermediate.IntermediateRes, gci)
			}
		default:
			// no rule applicable in phase one and not in normal form
			intermediate.IntermediateRes = append(intermediate.IntermediateRes, gci)
		}
	case BottomConcept:
		// rule NF4: Nothing to be done
	case Conjunction:
		// there are four cases here:
		// gci is of the form C1 ⊓ C2 ⊑ D (already in normal form)
		// or rule NF2 can be applied that is one of the concepts on the lhs
		// is in BCD and the ohter is not
		// or neither of these cases, in this case no rule can be applied
		firstBCD := lhs.C.IsInBCD()
		secondBCD := lhs.D.IsInBCD()
		switch {
		case firstBCD && secondBCD && gci.D.IsInBCD():
			// in normal form, so add it
			normalized := NewNormalizedCI(lhs.C, lhs.D, gci.D)
			intermediate.IntermediateCIs = append(intermediate.IntermediateCIs, normalized)
		case !firstBCD:
			// apply rule NF2
			newName := NewNamedConcept(intermediate.Distributor.Next())
			newConjunction := NewConjunction(lhs.D, newName)
			first := NewGCIConstraint(lhs.C, newName)
			second := NewGCIConstraint(newConjunction, gci.D)
			intermediate.Waiting = append(intermediate.Waiting, first, second)
		case !secondBCD:
			// apply rule NF2
			newName := NewNamedConcept(intermediate.Distributor.Next())
			newConjunction := NewConjunction(lhs.C, newName)
			first := NewGCIConstraint(lhs.D, newName)
			second := NewGCIConstraint(newConjunction, gci.D)
			intermediate.Waiting = append(intermediate.Waiting, first, second)
		default:
			// no rule can be applied in phase one
			intermediate.IntermediateRes = append(intermediate.IntermediateRes, gci)
		}
	case ExistentialConcept:
		// try to apply rule NF3, this is only possible the concept of the
		// existential concept is not in BCD
		if lhs.C.IsInBCD() {
			if BCDOrFalse(gci.D) {
				// in normal form, append to result
				normalized := NewNormalizedCILeftEx(lhs.R, lhs.C, gci.D)
				intermediate.IntermediateCILeft = append(intermediate.IntermediateCILeft, normalized)
			} else {
				// can't apply rule, add for further processing
				intermediate.IntermediateRes = append(intermediate.IntermediateRes, gci)
			}
		} else {
			// apply rule and add new elements
			newName := NewNamedConcept(intermediate.Distributor.Next())
			first := NewGCIConstraint(lhs.C, newName)
			newExistential := NewExistentialConcept(lhs.R, newName)
			second := NewGCIConstraint(newExistential, gci.D)
			intermediate.Waiting = append(intermediate.Waiting, first, second)
			// TODO(Fabian): Can we append second to the IntermediateRes already?
			// it should never be affected again in phase one?
		}
	}
}

// PhaseOneSingle runs phase 1 of the normalization for a given gci.
// That is it exhaustively applies the rules NF2 - NF4 and returns the
// result.
func PhaseOneSingle(gci *GCIConstraint, distributor *IntDistributor) *PhaseOneResult {
	res := NewPhaseOneResult(distributor)

	res.Waiting = []*GCIConstraint{gci}
	for len(res.Waiting) > 0 {
		n := len(res.Waiting)
		next := res.Waiting[n-1]
		res.Waiting[n-1] = nil
		res.Waiting = res.Waiting[:n-1]
		NormalizeStepPhaseOne(next, res)
	}
	return res
}

// NormalizeStepPhaseTwo performs one step of the normalization in phase 2.
// It tests what form the gci has and adds new formulae to the right slice
// of the intermediate result. That is it appends formulae to waiting
// or one of the intermediate results.
//
// Phase one means to apply the rules NF5 - NF7.
func NormalizeStepPhaseTwo(gci *GCIConstraint, intermediate *PhaseTwoResult) {
	// again a analyze what form we have
	// check which of the rules of phase two can be applied, if none can be
	// applied gci must already be in normal form
	if !gci.C.IsInBCD() && !gci.D.IsInBCD() {
		// apply NF5
		newName := NewNamedConcept(intermediate.Distributor.Next())
		first := NewGCIConstraint(gci.C, newName)
		second := NewGCIConstraint(newName, gci.D)
		intermediate.Waiting = append(intermediate.Waiting, first, second)
		return
	}
	switch rhs := gci.D.(type) {
	case ExistentialConcept:
		if rhs.C.IsInBCD() {
			// add result, now lhs must be in bcd, otherwise there was some mistake...
			// TODO remove this test once everything has been tested 1000 times
			if !gci.C.IsInBCD() {
				log.Printf("Found existential restriction on RHS, but the LHS is not in BCD (type %v) in normalization phase 2. MISTAKE\n",
					reflect.TypeOf(gci.C))
				return
			} else {
				// everything ok, add the result
				normalized := NewNormalizedCIRightEx(gci.C, rhs.R, rhs.C)
				intermediate.IntermediateCIRight = append(intermediate.IntermediateCIRight, normalized)
			}
		} else {
			// apply NF6
			newName := NewNamedConcept(intermediate.Distributor.Next())
			newExistential := NewExistentialConcept(rhs.R, newName)
			first := NewGCIConstraint(gci.C, newExistential)
			second := NewGCIConstraint(newName, rhs.C)
			intermediate.Waiting = append(intermediate.Waiting, first, second)
		}
	case Conjunction:
		// aply NF7
		first := NewGCIConstraint(gci.C, rhs.C)
		second := NewGCIConstraint(gci.C, rhs.D)
		intermediate.Waiting = append(intermediate.Waiting, first, second)
	default:
		// none of the rules can be applied in phase two, so it must be in normal
		// form and we can simply add the result
		// check for the type of the normal form

		// first case: lhs is in BCD and rhs is in BCD or false
		if gci.C.IsInBCD() && BCDOrFalse(gci.D) {
			normalized := NewNormalizedCI(gci.C, nil, gci.D)
			intermediate.IntermediateCIs = append(intermediate.IntermediateCIs, normalized)
			return
		}

		// second case: lhs is a concjunction, both C1 and C2 (lhs) must be in BCD
		// and D (rhs) must be in BCD or false
		if lhs, ok := gci.C.(Conjunction); ok && BCDOrFalse(gci.D) {
			// now both parts of the conjunction must be in BCD, check for this
			if lhs.C.IsInBCD() && lhs.D.IsInBCD() {
				normalized := NewNormalizedCI(lhs.C, lhs.D, rhs)
				intermediate.IntermediateCIs = append(intermediate.IntermediateCIs, normalized)
				return
			} else {
				fmt.Printf("Normalization phase two: Found conjunction on LHS, but its parts are not both in the BCD, types: %v and %v\n",
					reflect.TypeOf(lhs.C), reflect.TypeOf(lhs.D))
				return
			}
		}

		// third case: existential on rhs, this must not be checked because we
		// checked it before

		// foruth case: existential on lhs, the concept must be in BCD and the
		// rhs must be in BCD or false
		if lhs, ok := gci.C.(ExistentialConcept); ok && BCDOrFalse(gci.D) {
			// now check if condition is satisfied again
			if lhs.C.IsInBCD() {
				// TODO bad, don't remeber why...
				normalized := NewNormalizedCILeftEx(lhs.R, lhs.C, gci.D)
				intermediate.IntermediateCILeft = append(intermediate.IntermediateCILeft, normalized)
				return
			} else {
				log.Printf("Normalization phase two: Found existential on LHS, but its concept is not in BCD (type %v): MISTAKE\n",
					reflect.TypeOf(lhs.C))
				return
			}
		}

		// if we reach this point something went wrong: not in normal form
		log.Printf("Normalization phase two: Found invalid GCI %v. Types: %v and %v\n",
			gci, reflect.TypeOf(gci.C), reflect.TypeOf(gci.D))
	}
}

// PhaseTwoSingle runs phase 2 of the normalization for a given gci.
// That is it exhaustively applies the rules NF5 - NF7 and returns the
// result.
func PhaseTwoSingle(gci *GCIConstraint, distributor *IntDistributor) *PhaseTwoResult {
	res := NewPhaseTwoResult(distributor)

	res.Waiting = []*GCIConstraint{gci}
	for len(res.Waiting) > 0 {
		n := len(res.Waiting)
		next := res.Waiting[n-1]
		res.Waiting[n-1] = nil
		res.Waiting = res.Waiting[:n-1]
		NormalizeStepPhaseTwo(next, res)
	}
	return res
}

// NormalizeSingleRI normalizes a rule inclusion, that is it exhaustively
// applies rule NF1.
// firstNewIndex must be the next index we can use to create new roles.
// If k is the length of the left hand side it requires k - 2 new role names
// if k > 2 and 0 new names otherwise.
func NormalizeSingleRI(ri *RoleInclusion, firstNewIndex uint) []*NormalizedRI {
	lhs := ri.LHS
	n := uint(len(lhs))
	switch n {
	case 0:
		// TODO is it possible that the lhs is empty?
		return []*NormalizedRI{NewNormalizedRI(NoRole, NoRole, ri.RHS)}
	case 1:
		return []*NormalizedRI{NewNormalizedRI(lhs[0], NoRole, ri.RHS)}
	case 2:
		return []*NormalizedRI{NewNormalizedRI(lhs[0], lhs[1], ri.RHS)}
	default:
		// TODO thoroughly test this case
		rList := make([]*NormalizedRI, 0, n-1)
		// now we need a loop to add all RIs as given in the algorithm
		// add first one by hand
		// the first one always has the following form:
		// u1 o rk ⊑ s where u1 is a new role
		rList = append(rList, NewNormalizedRI(NewRole(firstNewIndex), lhs[n-1], ri.RHS))

		var i uint
		for i = 0; i < n-3; i++ {
			rList = append(rList, NewNormalizedRI(
				NewRole(firstNewIndex+i+1),
				lhs[n-2-i],
				NewRole(firstNewIndex+i)))
		}
		// also add the following RI which is left after the loop:
		// r1 o r2 ⊑ un
		// where un is the last new role
		rList = append(rList, NewNormalizedRI(lhs[0],
			lhs[1],
			NewRole(firstNewIndex+n-2-1)))
		return rList
	}
}

// TBoxNormalformBuilder is anything that transforms a tbox to a normalized
// tbox. Multiple calls to Normalize should be possible (i.e. data must be
// reset when running this method).
type TBoxNormalformBuilder interface {
	Normalize(tbox *TBox) *NormalizedTBox
}

// DefaultNormalFormBuilder is an implementation of TBoxNormalformBuilder
// that normalizes up to k formulae concurrently.
type DefaultNormalFormBuilder struct {
	k                  int
	rolesAfterPhaseOne uint
	distributor        *IntDistributor
	phaseOne           *PhaseOneResult
	ris                []*NormalizedRI
}

// NewDefaultNormalFormBUilder returns a new DefaultNormalFormBuilder.
// k must be a value > 0 and is the number of formulae that will be
// normalized concurrently. That is at most k normalizations will be performed
// at the same time. So k = 1 means that no concurrency happens, instead
// each formulae is normalized on it's own.
func NewDefaultNormalFormBUilder(k int) *DefaultNormalFormBuilder {
	return &DefaultNormalFormBuilder{k: k, phaseOne: NewPhaseOneResult(nil)}
}

// reset clears all data from the previous run.
func (builder *DefaultNormalFormBuilder) reset() {
	builder.rolesAfterPhaseOne = 0
	builder.distributor = nil
	builder.phaseOne = NewPhaseOneResult(nil)
	builder.ris = nil
}

// runPhaseOne normalizes all GCIs and RIs of the tbox (only rules NF1 - NF4).
// As described above at most k normalizations will happen concurrently.
func (builder *DefaultNormalFormBuilder) runPhaseOne(tbox *TBox) {
	distributor := NewIntDistributor(tbox.Components.Names)
	builder.distributor = distributor
	// create a channel that is used to coordinate the workers
	// this means: create a channel with buffer size k, writes to this channel
	// will block once k things are running, when a normlization is done
	// it will read a value from the channel
	workers := make(chan struct{}, builder.k)
	// a channel that is used to collect all normalization results for a single
	// formula
	resChan := make(chan *PhaseOneResult)

	// a channel for normalized RIs, the idea is the same as with resChan
	riChan := make(chan []*NormalizedRI)

	// a wait group to wait for everything to finish
	var wg sync.WaitGroup
	wg.Add(2)

	// start a listener that waits for all phase one results and merges
	// the current result with the new result
	go func() {
		defer wg.Done()
		for i := 0; i < len(tbox.GCIs); i++ {
			next := <-resChan
			builder.phaseOne.Union(next)
		}
	}()

	// listener for the ris
	go func() {
		defer wg.Done()
		for i := 0; i < len(tbox.RIs); i++ {
			next := <-riChan
			builder.ris = append(builder.ris, next...)
		}
	}()

	// start the GCI normalization
	for _, gci := range tbox.GCIs {
		// wait for a free worker
		workers <- struct{}{}
		// run normalization concurrently, free worker when done
		go func(gci *GCIConstraint) {
			resChan <- PhaseOneSingle(gci, distributor)
			<-workers
		}(gci)
	}

	// iterate over all ris and normalize them
	// this variable contains the next free index the normalization
	// can use to create new roles
	// the first one that is safe to use is the number of roles used in the
	// original model
	nextNew := tbox.Components.Roles
	for _, ri := range tbox.RIs {
		// wait for a free worker
		workers <- struct{}{}
		// run normalization concurrently
		go func(ri *RoleInclusion, nextIndex uint) {
			// TODO
			riChan <- NormalizeSingleRI(ri, nextIndex)
			<-workers
		}(ri, nextNew)
		// an RI of the form r1 o ... rk [= s requires k - 2 new role names if
		// k > 2
		// otherwise it requires 0
		k := uint(len(ri.LHS))
		if k > 2 {
			nextNew += (k - 2)
		}
	}
	builder.rolesAfterPhaseOne = nextNew
	wg.Wait()
}

// runPhaseTwo normalizes all GCIs of the tbox (rules NF5 - NF7).
// As described above at most k normalizations will happen concurrently.
func (builder *DefaultNormalFormBuilder) runPhaseTwo() *PhaseTwoResult {
	res := NewPhaseTwoResult(builder.distributor)
	res.IntermediateCILeft = builder.phaseOne.IntermediateCILeft
	res.IntermediateCIRight = builder.phaseOne.IntermediateCIRight
	res.IntermediateCIs = builder.phaseOne.IntermediateCIs
	// create a channel that is used to coordinate the workers
	// this means: create a channel with buffer size k, writes to this channel
	// will block once k things are running, when a normlization is done
	// it will read a value from the channel
	workers := make(chan struct{}, builder.k)
	// a channel that is used to collect all normalization results for a single
	// formula
	resChan := make(chan *PhaseTwoResult)

	// a channel to wait for everything to finish
	done := make(chan struct{})

	// start a listener for that waits for all phase two results and merges
	// the current result with the new result
	go func() {
		for i := 0; i < len(builder.phaseOne.IntermediateRes); i++ {
			next := <-resChan
			res.Union(next)
		}
		done <- struct{}{}
	}()

	// start the GCI normalization
	for _, gci := range builder.phaseOne.IntermediateRes {
		// wait for a free worker
		workers <- struct{}{}
		// run normalization concurrently, free worker when done
		go func(gci *GCIConstraint) {
			resChan <- PhaseTwoSingle(gci, builder.distributor)
			<-workers
		}(gci)
	}

	<-done
	return res
}

func (builder *DefaultNormalFormBuilder) Normalize(tbox *TBox) *NormalizedTBox {
	// run phase one
	builder.runPhaseOne(tbox)
	// now rune phase two
	phaseTwoRes := builder.runPhaseTwo()
	// create the updated components
	newComponents := NewELBaseComponents(tbox.Components.Nominals,
		tbox.Components.CDExtensions,
		// TODO should be correct?
		builder.distributor.Next(),
		builder.rolesAfterPhaseOne)
	res := NormalizedTBox{
		Components: newComponents,
		CIs:        phaseTwoRes.IntermediateCIs,
		CIRight:    phaseTwoRes.IntermediateCIRight,
		CILeft:     phaseTwoRes.IntermediateCILeft,
		RIs:        builder.ris,
	}
	// empty all fields, just to be sure
	// this may help garbage collection if we already delete all stuff
	builder.reset()
	return &res
}
