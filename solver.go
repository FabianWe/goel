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
	"sync"
)

// TODO redefine set interfaces s.t. that accept the components as well

type BCSet struct {
	m map[uint]struct{}
	c *ELBaseComponents
}

func NewBCSet(c *ELBaseComponents, initialCapacity uint) *BCSet {
	return &BCSet{
		m: make(map[uint]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCSet) Contains(c Concept) bool {
	_, has := s.m[c.NormalizedID(s.c)]
	return has
}

func (s *BCSet) Add(c Concept) bool {
	oldLen := len(s.m)
	s.m[c.NormalizedID(s.c)] = struct{}{}
	return oldLen != len(s.m)
}

func (s *BCSet) Union(other *BCSet) bool {
	oldLen := len(s.m)
	for v, _ := range other.m {
		s.m[v] = struct{}{}
	}
	return oldLen != len(s.m)
}

type bcPair struct {
	First, Second uint
}

type BCPairSet struct {
	m map[bcPair]struct{}
	c *ELBaseComponents
}

func NewBCPairSet(c *ELBaseComponents, initialCapacity uint) *BCPairSet {
	return &BCPairSet{
		m: make(map[bcPair]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCPairSet) Contains(c, d Concept) bool {
	_, has := s.m[bcPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}]
	return has
}

func (s *BCPairSet) Add(c, d Concept) bool {
	oldLen := len(s.m)
	p := bcPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}
	s.m[p] = struct{}{}
	return oldLen != len(s.m)
}

func (s *BCPairSet) AddID(c, d uint) bool {
	oldLen := len(s.m)
	p := bcPair{c, d}
	s.m[p] = struct{}{}
	return oldLen != len(s.m)
}

// Functions that check if a rule is applicable.

func CheckCR1(gci *NormalizedCI, sc *BCSet) bool {
	// TODO remove once tested
	if gci.C2 != nil {
		log.Printf("Invalid gci for CR1, must be of the form C ⊑ D, but got C1 ⊓ C2 ⊑ D: %v\n", gci)
		return false
	}
	return sc.Contains(gci.C1)
}

func CheckCR2(gci *NormalizedCI, sc *BCSet) bool {
	// TODO remove once tested
	if gci.C2 == nil {
		log.Printf("Invalid gci for CR1, must be of the form C1 ⊓ C2 ⊑ D, but got C ⊑ D: %v\n", gci)
		return false
	}
	return sc.Contains(gci.C1) && sc.Contains(gci.C2)
}

func CheckCR3(gci *NormalizedCIRightEx, sc *BCSet) bool {
	return sc.Contains(gci.C1)
}

func CheckCR4(gci *NormalizedCILeftEx, c, d Concept, sd *BCSet, sr *BCPairSet) bool {
	return sr.Contains(c, d) && sd.Contains(gci.C1)
}

func CheckCR5(c, d Concept, sd *BCSet, sr *BCPairSet) bool {
	return sd.Contains(Bottom) && sr.Contains(c, d)
}

func CheckCR6(a NominalConcept, c, d Concept, sc, sd *BCSet, search *GraphSearcher, components *ELBaseComponents) bool {
	return sc.Contains(a) && sd.Contains(a) && search.Search(c.NormalizedID(components), d.NormalizedID(components))
}

func CheckCR10(sr BCPairSet, c, d Concept) bool {
	return sr.Contains(c, d)
}

func CheckCR11(sr1, sr2 *BCPairSet, c, d, e Concept) bool {
	return sr1.Contains(c, d) && sr2.Contains(d, e)
}

// TODO remove map interfaces? not required I guess...

type NaiveSolver struct {
	S     []*BCSet
	R     []*BCPairSet
	graph ConceptGraph
}

func NewNaiveSolver(graph ConceptGraph) *NaiveSolver {
	return &NaiveSolver{
		S:     nil,
		R:     nil,
		graph: graph,
	}
}

func (solver *NaiveSolver) init(c *ELBaseComponents) {
	// initialize stuff, we do that concurrently
	var wg sync.WaitGroup
	wg.Add(3)
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	numBCD := c.NumBCD() + 1
	fmt.Printf("Got %d BCD elements and %d roles\n", numBCD, c.Roles)
	go func() {
		solver.graph.Init(numBCD)
		wg.Done()
	}()
	go func() {
		solver.S = make([]*BCSet, numBCD)
		// strictly speaking the bottom concept is not part of this and so
		// will be ignored
		// for the top concept we initialize S(⊤) = {⊤}
		// and for all other concepts C we initialize S(C) = {C, ⊤}
		solver.S[1] = NewBCSet(c, 10)
		solver.S[1].Add(Top)
		var i uint = 2
		for ; i < numBCD; i++ {
			solver.S[i] = NewBCSet(c, 10)
			solver.S[i].Add(Top)
			solver.S[i].Add(c.GetConcept(i))
		}
		wg.Done()
	}()
	go func() {
		solver.R = make([]*BCPairSet, c.Roles)
		var i uint = 0
		for ; i < c.Roles; i++ {
			solver.R[i] = NewBCPairSet(c, 10)
		}
		wg.Done()
	}()
	wg.Wait()
}

func (solver *NaiveSolver) updateR(r Role, c, d Concept, bc *ELBaseComponents) bool {
	cID, dID := c.NormalizedID(bc), d.NormalizedID(bc)
	if solver.R[uint(r)].AddID(cID, dID) {
		// update graph as well
		solver.graph.AddEdge(cID, dID)
		return true
	}
	return false
}

func (solver *NaiveSolver) Solve(tbox *NormalizedTBox) {
	solver.init(tbox.Components)
	changed := true
	for changed {
		changed = false
		// no try to apply each rule, if anything changes set changed to true
		// first try to apply rules CR1 - CR4, for each cgi there is only one
		// rule we can apply here
		// so first we iterate over all cgis of the form C1 ⊑ D and C1 ⊓ C2 ⊑ D
		for _, gci := range tbox.CIs {
			if gci.C2 == nil {
				// try CR1
				for _, sc := range solver.S[1:] {
					if CheckCR1(gci, sc) {
						// add to sc
						if sc.Add(gci.D) {
							changed = true
						}
					}
				}
			} else {
				// try CR2
				for _, sc := range solver.S[1:] {
					if CheckCR2(gci, sc) {
						// add to sc
						if sc.Add(gci.D) {
							changed = true
						}
					}
				}
			}
		}
		// now try rule CR3
		for c, gci := range tbox.CIRight {
			for _, sc := range solver.S[1:] {
				if CheckCR3(gci, sc) {
					// add
					r := gci.R
					conceptC, conceptD := tbox.Components.GetConcept(uint(c)), gci.C2
					if solver.updateR(r, conceptC, conceptD, tbox.Components) {
						changed = true
					}
				}
			}
		}
		// now try rule CR4
		// don't use CheckCR4, this way is faster
		for _, gci := range tbox.CILeft {
			// get the set R(r)
			sr := solver.R[uint(gci.R)]
			// iterate each pair (C, D) in R(r), then only check if D'
			// is in S(D)
			for p, _ := range sr.m {
				sd := solver.S[p.Second]
				if sd.Contains(gci.C1) {
					// add
					sc := solver.S[p.First]
					if sc.Add(gci.D) {
						changed = true
					}
				}
			}
		}
		// now try rule CR5
		for _, sr := range solver.R {
			// iterate over each pair (C, D) in R(r) and test if ⊥ is in S(D)
			for p, _ := range sr.m {
				sd := solver.S[p.Second]
				if sd.Contains(Bottom) {
					sc := solver.S[p.First]
					if sc.Add(Bottom) {
						changed = true
					}
				}
			}
		}
		// now try rule CR6
		// iterate over each nominal
		var nextNominal uint = 0
		// bottom concept has a place in S, but is not a part of it (see init)
		// thus we wish to remove the first element from solver.S
		s := solver.S[1:]
		for ; nextNominal < tbox.Components.Nominals; nextNominal++ {
			nominal := NewNominalConcept(nextNominal)
			// iterate over each S(C) and S(D)
			for i, sc := range s {
				if sc.Contains(nominal) {
					for _, sd := range s[i+1:] {
						if sd.Contains(nominal) {
							// now extend
						}
					}
				}
			}
		}
	}
}
