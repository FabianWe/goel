// The MIT License (MIT)

// Copyright (c) 2016, 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

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

type IntDistributor struct {
	next  uint
	mutex *sync.Mutex
}

func NewIntDistributor(next uint) *IntDistributor {
	return &IntDistributor{next: next, mutex: new(sync.Mutex)}
}

func (dist *IntDistributor) Next() uint {
	dist.mutex.Lock()
	defer dist.mutex.Unlock()
	next := dist.next
	dist.next++
	return next
}

type phaseOneResult struct {
	distributor     *IntDistributor
	waiting         []*GCIConstraint
	intermediateRes []*GCIConstraint
	// for CIs already in normal form we can just add them already for phase 2
	intermediateCIs     []*NormalizedCI
	intermediateCILeft  []*NormalizedCILeftEx
	intermediateCIRight []*NormalizedCIRightEx
}

func newPhaseOneResult(distributor *IntDistributor) *phaseOneResult {
	return &phaseOneResult{distributor: distributor}
}

func (res *phaseOneResult) union(other *phaseOneResult) {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		res.intermediateRes = append(res.intermediateRes, other.intermediateRes...)
	}()

	go func() {
		defer wg.Done()
		res.intermediateCIs = append(res.intermediateCIs, other.intermediateCIs...)
	}()

	go func() {
		defer wg.Done()
		res.intermediateCILeft = append(res.intermediateCILeft, other.intermediateCILeft...)
	}()

	go func() {
		defer wg.Done()
		res.intermediateCIRight = append(res.intermediateCIRight, other.intermediateCIRight...)
	}()

	wg.Wait()
}

type phaseTwoResult struct {
	distributor         *IntDistributor
	waiting             []*GCIConstraint
	intermediateCIs     []*NormalizedCI
	intermediateCILeft  []*NormalizedCILeftEx
	intermediateCIRight []*NormalizedCIRightEx
}

func newPhaseTwoResult(distributor *IntDistributor) *phaseTwoResult {
	return &phaseTwoResult{distributor: distributor}
}

func (res *phaseTwoResult) union(other *phaseTwoResult) {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.intermediateCIs = append(res.intermediateCIs, other.intermediateCIs...)
	}()

	go func() {
		defer wg.Done()
		res.intermediateCILeft = append(res.intermediateCILeft, other.intermediateCILeft...)
	}()

	go func() {
		defer wg.Done()
		res.intermediateCIRight = append(res.intermediateCIRight, other.intermediateCIRight...)
	}()

	wg.Wait()
}

func NormalizeStepPhaseOne(gci *GCIConstraint, intermediate *phaseOneResult) {
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
			intermediate.intermediateCIs = append(intermediate.intermediateCIs, normalized)
		case ExistentialConcept:
			// check if the rhs is of the form ∃r.C2 where C2 is in BCD
			if rhs.C.IsInBCD() {
				// in normal form, add it
				normalized := NewNormalizedCIRightEx(lhs, rhs.R, rhs.C)
				intermediate.intermediateCIRight = append(intermediate.intermediateCIRight, normalized)
			} else {
				// not in normal form
				intermediate.intermediateRes = append(intermediate.intermediateRes, gci)
			}
		default:
			// no rule applicable in phase one and not in normal form
			intermediate.intermediateRes = append(intermediate.intermediateRes, gci)
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
			intermediate.intermediateCIs = append(intermediate.intermediateCIs, normalized)
		case !firstBCD:
			// apply rule NF2
			newName := NewNamedConcept(intermediate.distributor.Next())
			newConjunction := NewConjunction(lhs.D, newName)
			first := NewGCIConstraint(lhs.C, newName)
			second := NewGCIConstraint(newConjunction, gci.D)
			intermediate.waiting = append(intermediate.waiting, first, second)
		case !secondBCD:
			// apply rule NF2
			newName := NewNamedConcept(intermediate.distributor.Next())
			newConjunction := NewConjunction(lhs.C, newName)
			first := NewGCIConstraint(lhs.D, newName)
			second := NewGCIConstraint(newConjunction, gci.D)
			intermediate.waiting = append(intermediate.waiting, first, second)
		default:
			// no rule can be applied in phase one
			intermediate.intermediateRes = append(intermediate.intermediateRes, gci)
		}
	case ExistentialConcept:
		// try to apply rule NF3, this is only possible the concept of the
		// existential concept is not in BCD
		if lhs.C.IsInBCD() {
			if BCDOrFalse(gci.D) {
				// in normal form, append to result
				normalized := NewNormalizedCILeftEx(lhs.R, lhs.C, gci.D)
				intermediate.intermediateCILeft = append(intermediate.intermediateCILeft, normalized)
			} else {
				// can't apply rule, add for further processing
				intermediate.intermediateRes = append(intermediate.intermediateRes, gci)
			}
		} else {
			// apply rule and add new elements
			newName := NewNamedConcept(intermediate.distributor.Next())
			first := NewGCIConstraint(lhs.C, newName)
			newExistential := NewExistentialConcept(lhs.R, newName)
			second := NewGCIConstraint(newExistential, gci.D)
			intermediate.waiting = append(intermediate.waiting, first, second)
			// TODO(Fabian): Can we append second to the IntermediateRes already?
			// it should never be affected again in phase one?
		}
	}
}

func NormalizeStepPhaseTwo(gci *GCIConstraint, intermediate *phaseTwoResult) {
	// again a analyze what form we have
	// check which of the rules of phase two can be applied, if none can be
	// applied gci must already be in normal form
	if !gci.C.IsInBCD() && !gci.D.IsInBCD() {
		// apply NF5
		newName := NewNamedConcept(intermediate.distributor.Next())
		first := NewGCIConstraint(gci.C, newName)
		second := NewGCIConstraint(newName, gci.D)
		intermediate.waiting = append(intermediate.waiting, first, second)
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
				intermediate.intermediateCIRight = append(intermediate.intermediateCIRight, normalized)
			}
		} else {
			// apply NF6
			newName := NewNamedConcept(intermediate.distributor.Next())
			newExistential := NewExistentialConcept(rhs.R, newName)
			first := NewGCIConstraint(gci.C, newExistential)
			second := NewGCIConstraint(newName, rhs.C)
			intermediate.waiting = append(intermediate.waiting, first, second)
		}
	case Conjunction:
		// aply NF7
		first := NewGCIConstraint(gci.C, rhs.C)
		second := NewGCIConstraint(gci.C, rhs.D)
		intermediate.waiting = append(intermediate.waiting, first, second)
	default:
		// none of the rules can be applied in phase two, so it must be in normal
		// form and we can simply add the result
		// check for the type of the normal form

		// first case: lhs is in BCD and rhs is in BCD or false
		if gci.C.IsInBCD() && BCDOrFalse(gci.D) {
			normalized := NewNormalizedCI(gci.C, nil, gci.D)
			intermediate.intermediateCIs = append(intermediate.intermediateCIs, normalized)
			return
		}

		// second case: lhs is a concjunction, both C1 and C2 (lhs) must be in BCD
		// and D (rhs) must be in BCD or false
		if lhs, ok := gci.C.(Conjunction); ok && BCDOrFalse(gci.D) {
			// now both parts of the conjunction must be in BCD, check for this
			if lhs.C.IsInBCD() && lhs.D.IsInBCD() {
				normalized := NewNormalizedCI(lhs.C, lhs.D, rhs)
				intermediate.intermediateCIs = append(intermediate.intermediateCIs, normalized)
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
				intermediate.intermediateCILeft = append(intermediate.intermediateCILeft, normalized)
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

type TBoxNormalformBuilder interface {
	Normalize(tbox *TBox) *NormalizedTBox
}
