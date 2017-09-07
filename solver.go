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

type BCSet interface {
	Contains(c Concept) bool
	Add(c Concept) bool
}

type BCSetFactory func() BCSet

type MapBCSet struct {
	m map[uint]struct{}
	c *ELBaseComponents
}

func NewMapBCSet(c *ELBaseComponents, initialCapacity uint) *MapBCSet {
	return &MapBCSet{
		m: make(map[uint]struct{}, initialCapacity),
		c: c,
	}
}

func MapBCSetFactory(c *ELBaseComponents, initialCapacity uint) BCSetFactory {
	return func() BCSet {
		return NewMapBCSet(c, initialCapacity)
	}
}

func (s *MapBCSet) Contains(c Concept) bool {
	_, has := s.m[c.NormalizedID(s.c)]
	return has
}

// TODO avoid lookups, simply check length?
func (s *MapBCSet) Add(c Concept) bool {
	normalizedID := c.NormalizedID(s.c)
	if _, has := s.m[normalizedID]; has {
		return false
	} else {
		s.m[normalizedID] = struct{}{}
		return true
	}
}

type BCPairSet interface {
	Contains(c, d Concept) bool
	Add(c, d Concept) bool
}

type BCPairSetFactory func() BCPairSet

type bcPair struct {
	First, Second uint
}

func newBCPair(c, d Concept, comp *ELBaseComponents) bcPair {
	return bcPair{
		First:  c.NormalizedID(comp),
		Second: d.NormalizedID(comp),
	}
}

type MapBCPairSet struct {
	m map[bcPair]struct{}
	c *ELBaseComponents
}

func NewMapBCPairSet(c *ELBaseComponents, initialCapacity uint) *MapBCPairSet {
	return &MapBCPairSet{
		m: make(map[bcPair]struct{}, initialCapacity),
		c: c,
	}
}

func MapBCPairSetFactory(c *ELBaseComponents, initialCapacity uint) BCPairSetFactory {
	return func() BCPairSet {
		return NewMapBCPairSet(c, initialCapacity)
	}
}

func (s *MapBCPairSet) Contains(c, d Concept) bool {
	_, has := s.m[newBCPair(c, d, s.c)]
	return has
}

func (s *MapBCPairSet) Add(c, d Concept) bool {
	p := newBCPair(c, d, s.c)
	if _, has := s.m[p]; has {
		return false
	}
	s.m[p] = struct{}{}
	return true
}

type BCMap interface {
	GetVAlue(c Concept) BCSet
}

type DefaultBCMap struct {
	m       map[uint]BCSet
	factory BCSetFactory
	c       *ELBaseComponents
}

func NewDefaultBCMap(factory BCSetFactory, c *ELBaseComponents, initialCapacity uint) *DefaultBCMap {
	return &DefaultBCMap{
		m:       make(map[uint]BCSet, initialCapacity),
		factory: factory,
		c:       c,
	}
}

func (m *DefaultBCMap) GetValue(c Concept) BCSet {
	id := c.NormalizedID(m.c)
	if res, has := m.m[id]; has {
		return res
	} else {
		newSet := m.factory()
		m.m[id] = newSet
		return newSet
	}
}

type RoleMap interface {
	GetValue(r Role) BCPairSet
}

type DefaultRoleMap struct {
	m       map[Role]BCPairSet
	factory BCPairSetFactory
}

func NewDefaultRoleMap(factory BCPairSetFactory, initialCapacity uint) *DefaultRoleMap {
	return &DefaultRoleMap{
		m:       make(map[Role]BCPairSet, initialCapacity),
		factory: factory,
	}
}

func (m *DefaultRoleMap) GetValue(r Role) BCPairSet {
	if res, has := m.m[r]; has {
		return res
	} else {
		newSet := m.factory()
		m.m[r] = newSet
		return newSet
	}
}

// Functions that check if a rule is applicable.

func CheckCR1(gci *NormalizedCI, sc BCSet) bool {
	// TODO remove once tested
	if gci.C2 != nil {
		log.Printf("Invalid gci for CR1, must be of the form C ⊑ D, but got C1 ⊓ C2 ⊑ D: %v\n", gci)
		return false
	}
	return sc.Contains(gci.C1)
}

func CheckCR2(gci *NormalizedCI, sc BCSet) bool {
	// TODO remove once tested
	if gci.C2 == nil {
		log.Printf("Invalid gci for CR1, must be of the form C1 ⊓ C2 ⊑ D, but got C ⊑ D: %v\n", gci)
		return false
	}
	return sc.Contains(gci.C1) && sc.Contains(gci.C2)
}

func CheckCR3(gci *NormalizedCIRightEx, sc BCSet) bool {
	return sc.Contains(gci.C1)
}

func CheckCR4(gci *NormalizedCILeftEx, c, d Concept, sd BCSet, sr BCPairSet) bool {
	return sr.Contains(c, d) && sd.Contains(gci.C1)
}

func CheckCR5(c, d Concept, sd BCSet, sr BCPairSet) bool {
	return sd.Contains(Bottom) && sr.Contains(c, d)
}

func CheckCR6(a NominalConcept, c, d Concept, sc, sd BCSet, search *GraphSearcher, components *ELBaseComponents) bool {
	return sc.Contains(a) && sd.Contains(a) && search.Search(c.NormalizedID(components), d.NormalizedID(components))
}

func CheckCR10(sr BCPairSet, c, d Concept) bool {
	return sr.Contains(c, d)
}

func CheckCR11(sr1, sr2 BCPairSet, c, d, e Concept) bool {
	return sr1.Contains(c, d) && sr2.Contains(d, e)
}

// TODO remove map interfaces? not required I guess...

type NaiveSolver struct {
	S        []BCSet
	R        []BCPairSet
	graph    ConceptGraph
	sFactory BCSetFactory
	rFactory BCPairSetFactory
}

func NewNaiveSolver(sFactory BCSetFactory, rFactory BCPairSetFactory, graph ConceptGraph) *NaiveSolver {
	return &NaiveSolver{
		S:        nil,
		R:        nil,
		graph:    graph,
		sFactory: sFactory,
		rFactory: rFactory,
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
		solver.S = make([]BCSet, numBCD)
		// strictly speaking the bottom concept is not part of this and so
		// will be ignored
		// for the top concept we initialize S(⊤) = {⊤\
		// and for all other concepts C we initialize S(C) = {C, ⊤}
		solver.S[1] = solver.sFactory()
		solver.S[1].Add(Top)
		var i uint = 2
		for ; i < numBCD; i++ {
			solver.S[i] = solver.sFactory()
			solver.S[i].Add(Top)
			solver.S[i].Add(c.GetConcept(i))
		}
		wg.Done()
	}()
	go func() {
		solver.R = make([]BCPairSet, c.Roles)
		var i uint = 0
		for ; i < c.Roles; i++ {
			solver.R[i] = solver.rFactory()
		}
		wg.Done()
	}()
	wg.Wait()
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
					if solver.R[uint(r)].Add(tbox.Components.GetConcept(uint(c)), gci.C2) {
						changed = true
					}
				}
			}
		}
		// now try rule CR4
		// don't use CheckCR4, this way is faster
		for _, _ = range tbox.CILeft {
			// get the set R(r)
			// sr := solver.R[uint(gci.R)]
			// iterate each pair (C, D) in R(r), then only check if D'
			// is in S(D)
		}
	}
}
