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
	"fmt"
	"math/rand"
)

type RandomELBuilder struct {
	NumIndividuals     uint
	NumConceptNames    uint
	NumRoles           uint
	NumConcreteDomains uint
	MaxCDSize          uint
	MaxNumPredicates   uint
	MaxNumFeatures     uint
}

func (builder *RandomELBuilder) GenerateRandomTBox(numCDExtensions, numConjunctions,
	numExistentialRestrictions, maxtRILHS, numGCI, numRI uint) ([]Concept, *TBox) {
	baseComponents := NewELBaseComponents(builder.NumIndividuals, numCDExtensions, builder.NumConceptNames, builder.NumRoles)
	concepts := make([]Concept, 0)
	// first build all nominal concepts
	var i uint
	for ; i < builder.NumIndividuals; i++ {
		next := NewNominalConcept(i)
		concepts = append(concepts, next)
	}
	// next build all named concepts
	i = 0
	for ; i < builder.NumConceptNames; i++ {
		next := NewNamedConcept(i)
		concepts = append(concepts, next)
	}
	// create all CDExtensions
	i = 0
	for ; i < numCDExtensions; i++ {
		next := NewConcreteDomainExtension(i)
		concepts = append(concepts, next)
	}
	// we create conjunctions and existenstial restrictions in one loop, s.t.
	// they get mixed up a bit
	var currentConjunctions uint = 0
	var currentExistentials uint = 0
	for currentConjunctions < numConjunctions || currentExistentials < numExistentialRestrictions {
		// first decide whether to use a conjunction or an existential
		isNextConjunction := (rand.Intn(2) == 0)
		if isNextConjunction {
			// first check if we're still allowed to add another
			// conjunction, otherwise we must use an existential
			if currentConjunctions >= numConjunctions {
				isNextConjunction = false
			}
		} else {
			// we're to create an existensial, again check if we're allowed to do this
			if currentExistentials >= numExistentialRestrictions {
				isNextConjunction = true
			}
		}
		// now finally we can create the next concept dependend on isNextConjunction
		if isNextConjunction {
			// randomly select to concepts
			var c1 Concept
			c1 = concepts[rand.Intn(len(concepts))]
			var c2 Concept
			c2 = concepts[rand.Intn(len(concepts))]
			next := NewConjunction(c1, c2)
			concepts = append(concepts, next)
			currentConjunctions++
		} else {
			// choose role
			// TODO not so nice, but ok
			r := NewRole(uint(rand.Intn(int(builder.NumRoles))))
			// chose a concept
			c := concepts[rand.Intn(len(concepts))]
			next := NewExistentialConcept(r, c)
			concepts = append(concepts, next)
			currentExistentials++
		}
	}
	// build all ris
	ris := make([]*RoleInclusion, 0, numRI)
	i = 0
	for ; i < numRI; i++ {
		rhs := NewRole(uint(rand.Intn(int(builder.NumRoles))))
		lhsSize := rand.Intn(int(maxtRILHS))
		if lhsSize == 0 {
			lhsSize = 1
		}
		lhs := make([]Role, 0, lhsSize)
		for j := 0; j < lhsSize; j++ {
			// add random role
			nextRole := NewRole(uint(rand.Intn(int(builder.NumRoles))))
			lhs = append(lhs, nextRole)
		}
		ris = append(ris, NewRoleInclusion(lhs, rhs))
	}

	gcis := make([]*GCIConstraint, 0, numGCI)
	i = 0
	for ; i < numGCI; i++ {
		lhs := concepts[rand.Intn(len(concepts))]
		rhs := concepts[rand.Intn(len(concepts))]
		gcis = append(gcis, NewGCIConstraint(lhs, rhs))
	}
	return concepts, NewTBox(baseComponents, gcis, ris)
}

type NormalizedRandomELBuilder struct {
	NumIndividuals  uint
	NumConceptNames uint
	NumRoles        uint
}

func (builder *NormalizedRandomELBuilder) chooseConcept() uint {
	n := builder.NumIndividuals + builder.NumConceptNames
	return uint(rand.Intn(int(n)) + 2)
}

func (builder *NormalizedRandomELBuilder) chooseRole() uint {
	return uint(rand.Intn(int(builder.NumRoles)))
}

func (builder *NormalizedRandomELBuilder) GenerateRandomTBox(numNormalizedCI,
	numExistentialRestrictions, numRI int) *NormalizedTBox {
	fmt.Println("WHAT?!")

	c := NewELBaseComponents(builder.NumIndividuals, 0, builder.NumConceptNames, builder.NumRoles)

	cis := make([]*NormalizedCI, numNormalizedCI)
	for i := 0; i < int(numNormalizedCI); i++ {
		if rand.Intn(2) == 0 {
			// C2 is nil
			c1, d := builder.chooseConcept(), builder.chooseConcept()
			cis[i] = NewNormalizedCISingle(c.GetConcept(c1), c.GetConcept(d))
		} else {
			// not nil
			c1, c2, d := builder.chooseConcept(), builder.chooseConcept(), builder.chooseConcept()
			cis[i] = NewNormalizedCI(c.GetConcept(c1), c.GetConcept(c2), c.GetConcept(d))
		}
	}

	ciLeft := make([]*NormalizedCILeftEx, 0)
	ciRight := make([]*NormalizedCIRightEx, 0)
	for i := 0; i < int(numExistentialRestrictions); i++ {
		if rand.Intn(2) == 0 {
			// left
			c1, d, r := builder.chooseConcept(), builder.chooseConcept(), builder.chooseRole()
			ciLeft = append(ciLeft, NewNormalizedCILeftEx(NewRole(r), c.GetConcept(c1), c.GetConcept(d)))
		} else {
			// right
			c1, r, c2 := builder.chooseConcept(), builder.chooseRole(), builder.chooseConcept()
			ciRight = append(ciRight, NewNormalizedCIRightEx(c.GetConcept(c1), NewRole(r), c.GetConcept(c2)))
		}
	}
	ris := make([]*NormalizedRI, numRI)
	for i := 0; i < int(numRI); i++ {
		if rand.Intn(2) == 0 {
			// r2 is NoRole
			r1, s := builder.chooseRole(), builder.chooseRole()
			ris[i] = NewNormalizedRISingle(NewRole(r1), NewRole(s))
		} else {
			// r2 is some role
			r1, r2, s := builder.chooseRole(), builder.chooseRole(), builder.chooseRole()
			ris[i] = NewNormalizedRI(NewRole(r1), NewRole(r2), NewRole(s))
		}
	}
	return &NormalizedTBox{
		Components: c,
		CIs:        cis,
		CILeft:     ciLeft,
		CIRight:    ciRight,
		RIs:        ris,
	}
}
