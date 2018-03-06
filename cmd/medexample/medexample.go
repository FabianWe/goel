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

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/FabianWe/goel"
)

type stringMap map[uint]string

func (m stringMap) getValue(val uint) string {
	if val, has := m[val]; has {
		return val
	}
	log.Fatal("Looking up invalid id in map")
	return ""
}

func printMap(m stringMap, c uint, s *goel.BCSet) string {
	name := m.getValue(c)
	strs := make([]string, 0, len(s.M))
	for d, _ := range s.M {
		if dName, has := m[d]; has {
			strs = append(strs, dName)
		}
	}
	joined := strings.Join(strs, ", ")
	return fmt.Sprintf("%s subsumes {%s}", name, joined)
}

func main() {
	endocardium := goel.NewNamedConcept(0)
	tissue := goel.NewNamedConcept(1)
	heartWall := goel.NewNamedConcept(2)
	heartValve := goel.NewNamedConcept(3)
	bodyWall := goel.NewNamedConcept(4)
	heart := goel.NewNamedConcept(5)
	bodyValve := goel.NewNamedConcept(6)
	endocarditis := goel.NewNamedConcept(7)
	inflammation := goel.NewNamedConcept(8)
	disease := goel.NewNamedConcept(9)
	heartDisease := goel.NewNamedConcept(10)
	criticalDisease := goel.NewNamedConcept(11)

	actsOn := goel.NewRole(0)
	partOf := goel.NewRole(1)
	contIn := goel.NewRole(2)
	hasLoc := goel.NewRole(3)

	var conceptMap stringMap = make(map[uint]string)
	var roleMap stringMap = make(map[uint]string)
	c := goel.NewELBaseComponents(0, 0, 12, 4)
	conceptMap[endocardium.NormalizedID(c)] = "endocardium"
	conceptMap[tissue.NormalizedID(c)] = "tissue"
	conceptMap[heartWall.NormalizedID(c)] = "heartWall"
	conceptMap[heartValve.NormalizedID(c)] = "heartValve"
	conceptMap[bodyWall.NormalizedID(c)] = "bodyWall"
	conceptMap[heart.NormalizedID(c)] = "heart"
	conceptMap[bodyValve.NormalizedID(c)] = "bodyValve"
	conceptMap[endocarditis.NormalizedID(c)] = "endocarditis"
	conceptMap[inflammation.NormalizedID(c)] = "inflammation"
	conceptMap[disease.NormalizedID(c)] = "disease"
	conceptMap[heartDisease.NormalizedID(c)] = "heartDisease"
	conceptMap[criticalDisease.NormalizedID(c)] = "criticalDisease"

	roleMap[uint(actsOn)] = "acts-on"
	roleMap[uint(partOf)] = "partOf"
	roleMap[uint(contIn)] = "contIn"
	roleMap[uint(hasLoc)] = "hasLoc"

	gcis := make([]*goel.GCIConstraint, 0)
	ris := make([]*goel.RoleInclusion, 0)
	var lhs, rhs goel.Concept

	lhs = endocardium
	rhs = goel.NewMultiConjunction(tissue,
		goel.NewExistentialConcept(contIn, heartWall),
		goel.NewExistentialConcept(contIn, heartValve))
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = heartWall
	rhs = goel.NewMultiConjunction(heartWall,
		goel.NewExistentialConcept(partOf, heart))
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = heartValve
	rhs = goel.NewMultiConjunction(bodyValve,
		goel.NewExistentialConcept(partOf, heart))
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = endocarditis
	rhs = goel.NewMultiConjunction(inflammation,
		goel.NewExistentialConcept(hasLoc, endocardium))
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = inflammation
	rhs = goel.NewMultiConjunction(disease,
		goel.NewExistentialConcept(actsOn, tissue))
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = goel.NewMultiConjunction(heartDisease,
		goel.NewExistentialConcept(hasLoc, heartValve))
	rhs = criticalDisease
	gcis = append(gcis, goel.NewGCIConstraint(lhs, rhs))

	lhs = heartDisease
	rhs = goel.NewMultiConjunction(disease,
		goel.NewExistentialConcept(hasLoc, heart))
	gcis = append(gcis,
		goel.NewGCIConstraint(lhs, rhs),
		goel.NewGCIConstraint(rhs, lhs))

	var rlhs []goel.Role
	var rrhs goel.Role

	rlhs = []goel.Role{partOf, partOf}
	rrhs = partOf
	ris = append(ris, goel.NewRoleInclusion(rlhs, rrhs))

	rlhs = []goel.Role{partOf}
	rrhs = contIn
	ris = append(ris, goel.NewRoleInclusion(rlhs, rrhs))

	rlhs = []goel.Role{hasLoc, contIn}
	rrhs = hasLoc
	ris = append(ris, goel.NewRoleInclusion(rlhs, rrhs))

	// finally done!
	// create tbox, normalize it and run classification
	tbox := goel.NewTBox(c, gcis, ris)
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	normalized := normalizer.Normalize(tbox)

	// solve it with the concurrent classifier
	solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	solver.Solve(normalized)

	// now print some information we have gathered
	fmt.Println(len(solver.S))
	for i := 2; i < len(solver.S); i++ {
		if _, has := conceptMap[uint(i)]; has {
			fmt.Println(printMap(conceptMap, uint(i), solver.S[i]))
		}
	}
}
