// The MIT License (MIT)
//
// Copyright (c) 2016, 2017, 2018 Fabian Wenzelmann
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

	"github.com/FabianWe/goel/domains"
)

// TODO redefine set interfaces s.t. that accept the components as well

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

// func CheckCR6(a NominalConcept, c, d Concept, sc, sd *BCSet, search *GraphSearcher, components *ELBaseComponents) bool {
// 	return sc.Contains(a) && sd.Contains(a) && search.Search(c.NormalizedID(components), d.NormalizedID(components))
// }

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
	// set in init
	searcher     *GraphSearcher
	searchMethod ReachabilitySearch
}

func NewNaiveSolver(graph ConceptGraph, search ReachabilitySearch) *NaiveSolver {
	return &NaiveSolver{
		S:            nil,
		R:            nil,
		graph:        graph,
		searcher:     nil,
		searchMethod: search,
	}
}

func (solver *NaiveSolver) init(c *ELBaseComponents) {
	// initialize stuff, we do that concurrently
	var wg sync.WaitGroup
	wg.Add(4)
	// we use + 1 here because we want to use the normalized id directly, so
	// the bottom concept must be taken into consideration
	numBCD := c.NumBCD() + 1
	// fmt.Printf("Got %d BCD elements and %d roles\n", numBCD, c.Roles)
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
	go func() {
		solver.searcher = NewGraphSearcher(solver.searchMethod, c)
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

// TODO seems some iterations are just... stupid.
// check again!
func (solver *NaiveSolver) Solve(tbox *NormalizedTBox, manager *domains.CDManager) {
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
		for _, gci := range tbox.CIRight {
			for c, sc := range solver.S[1:] {
				if CheckCR3(gci, sc) {
					// add
					r := gci.R
					conceptC, conceptD := tbox.Components.GetConcept(uint(c+1)), gci.C2
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
			for p, _ := range sr.M {
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
			for p, _ := range sr.M {
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
		// TODO think this case through again!
		// iterate over each nominal
		var nextNominal uint = 0
		// bottom concept has a place in S, but is not a part of it (see init)
		// thus we wish to remove the first element from solver.S
		for ; nextNominal < tbox.Components.Nominals; nextNominal++ {
			nominal := NewNominalConcept(nextNominal)
			// iterate over each S(C) and S(D)
			// to do so we iterate over the id of each concept
			var i uint = 1
			// we start the search with i = 1 and iterate over all possible concept
			// ids
			for ; i < uint(len(solver.S)); i++ {
				sc := solver.S[i]
				if sc.Contains(nominal) {
					var j uint = i + 1
					for ; j < uint(len(solver.S)); j++ {
						sd := solver.S[j]
						if sd.Contains(nominal) {
							// check if C ↝ D
							// we also have to check if D ↝ C because we just check
							// all pairs (i, j) where j > i.
							// Thus we could have that i is not connected to j
							// but j is connected to i. In this case we must apply the rule!
							searchRes := solver.searcher.BidrectionalSearch(solver.graph, i, j)
							switch searchRes {
							case BidrectionalBoth:
								// update both
								// TODO could be done simpler because they should be equal
								// but well...
								if sc.Union(sd) {
									changed = true
								}
								if sd.Union(sc) {
									changed = true
								}
							case BidrectionalDirect:
								// update only first
								if sc.Union(sd) {
									changed = true
								}
							case BidrectionalReverse:
								if sd.Union(sc) {
									changed = true
								}
								// no default case, we simply do nothing
							}
						}
					}
				}
			}
		}
		// now try CR7 and CR8
		var i uint = 1
		// iterate over each c
		for ; i < uint(len(solver.S)); i++ {
			// get conjunction
			sc := solver.S[i]
			conjunctions := sc.GetCDConjunction(manager)
			if len(conjunctions) != 1 {
				panic("Only support for one concrete domain at the moment")
			}
			conjunction := conjunctions[0]
			// get domain
			domain := manager.GetDomainByID(0)
			// check if unsatisfiable, if yes apply CR7 and add false
			if !domain.ConjSat(conjunction...) {
				// a little bit nicer here than before by avoid the if...
				changed = sc.Add(Bottom) || changed
				// add all formulae from this domain because false implies everything
				// (rule CR8)
				// this must also include the new formula of course
				for _, formula := range manager.GetFormulaeFor(0) {
					formulaID := formula.FormulaID
					asExtension := NewConcreteDomainExtension(formulaID)
					// now add
					changed = sc.Add(asExtension) || changed
				}
			} else {
				// can't apply CR7 and we have to check CR8 for each formula
				for _, formula := range manager.GetFormulaeFor(0) {
					// check if the implication is true
					if domain.Implies(formula.Formula, conjunction...) {
						// add
						formulaID := formula.FormulaID
						asExtension := NewConcreteDomainExtension(formulaID)
						// add
						changed = sc.Add(asExtension) || changed
					}
				}
			}
		}
		// now try rule CR10 and CR11
		i = 0
		for ; i < uint(len(tbox.RIs)); i++ {
			ri := tbox.RIs[i]
			if ri.R2 == NoRole {
				// rule CR10
				r, s := ri.R1, ri.S
				// iterate over each pair in R(r)
				rr := solver.R[uint(r)]
				rs := solver.R[uint(s)]
				for pair, _ := range rr.M {
					if rs.AddID(pair.First, pair.Second) {
						changed = true
					}
				}
			} else {
				// rule CR11
				r1, r2, r3 := ri.R1, ri.R2, ri.S
				rr1 := solver.R[uint(r1)]
				rr2 := solver.R[uint(r2)]
				rr3 := solver.R[uint(r3)]
				// iterate over each pair (C1, D1) in R(r1)
				for pair1, _ := range rr1.M {
					// iterate over each pair (C2, D2) in R(r2) and check if D1 = C2
					for pair2, _ := range rr2.M {
						if pair1.Second == pair2.First {
							// add (C1, D2) to R(r3)
							if rr3.AddID(pair1.First, pair2.Second) {
								changed = true
							}
						}
					}
				}
			}
		}
	}
}
