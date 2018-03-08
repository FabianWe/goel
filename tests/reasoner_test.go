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

package tests

import (
	"testing"

	"github.com/FabianWe/goel"
	"github.com/FabianWe/goel/domains"
)

var ds *domains.CDManager = domains.NewCDManager()

func assertSContains(t *testing.T, ids []uint, s *goel.BCSet) {
	for _, id := range ids {
		if _, has := s.M[id]; !has {
			t.Errorf("Expected S(C) to contain %d, but it's not contained", id)
		}
	}
}

func assertRContains(t *testing.T, pairs []goel.BCPair, r *goel.Relation) {
	for _, pair := range pairs {
		first, second := pair.First, pair.Second
		if !r.Contains(first, second) {
			t.Errorf("Expected R(r) to contain (%d, %d), but it's not contained", first, second)
		}
	}
}

func runConcurrent(tbox *goel.TBox) *goel.ConcurrentSolver {
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	normalized := normalizer.Normalize(tbox)
	solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized, ds)
	solver.Solve(normalized)
	return solver
}

// TestTrees tests the simple ontology
// Maple ⊑ LeafTree ⊑ Tree ⊑ Plant
func TestTrees(t *testing.T) {
	maple := goel.NewNamedConcept(0)
	leafTree := goel.NewNamedConcept(1)
	tree := goel.NewNamedConcept(2)
	plant := goel.NewNamedConcept(3)

	c := goel.NewELBaseComponents(0, 0, 4, 0)

	gcis := []*goel.GCIConstraint{
		goel.NewGCIConstraint(maple, leafTree),
		goel.NewGCIConstraint(leafTree, tree),
		goel.NewGCIConstraint(tree, plant),
	}
	tbox := goel.NewTBox(c, gcis, nil)
	solver := runConcurrent(tbox)
	// now assert
	mapleID := maple.NormalizedID(c)
	leafTreeID := leafTree.NormalizedID(c)
	treeID := tree.NormalizedID(c)
	plantID := tree.NormalizedID(c)

	// assert that plant contains plant
	assertSContains(t, []uint{plantID}, solver.S[plantID])
	// assert tree contains tree and plant
	assertSContains(t, []uint{treeID, plantID}, solver.S[treeID])
	// assert leaf tree contains leaf tree, tree and plant
	assertSContains(t, []uint{leafTreeID, treeID, plantID}, solver.S[leafTreeID])
	// assert maple contains everything
	assertSContains(t, []uint{mapleID, leafTreeID, treeID, plantID}, solver.S[mapleID])
}

// from now on more technical examples, too lazy to think of something
// interesting.

// TestCR2 tests rules CR1 and CR2.
// A1 ⊑ A2 ⊑ A3, A1 ⊓ A2 ⊑ C4 ==> C4 ∈ S(A1)
func TestCR2(t *testing.T) {
	a1 := goel.NewNamedConcept(0)
	a2 := goel.NewNamedConcept(1)
	a3 := goel.NewNamedConcept(2)
	a4 := goel.NewNamedConcept(3)

	c := goel.NewELBaseComponents(0, 0, 4, 0)

	id1 := a1.NormalizedID(c)
	id2 := a2.NormalizedID(c)
	id3 := a3.NormalizedID(c)
	id4 := a4.NormalizedID(c)

	gcis := []*goel.GCIConstraint{
		goel.NewGCIConstraint(a1, a2),
		goel.NewGCIConstraint(a2, a3),
		goel.NewGCIConstraint(goel.NewConjunction(a1, a2), a4),
	}

	tbox := goel.NewTBox(c, gcis, nil)

	solver := runConcurrent(tbox)

	assertSContains(t, []uint{id1, id2, id3, id4}, solver.S[id1])
}

// TestCR3 tests rule CR3:
// A1 ⊑ A2, A2 ⊑ ∃r.C3 ==> R(r) contains (A1, A3) and (A2, A3)
func TestCR3(t *testing.T) {
	a1 := goel.NewNamedConcept(0)
	a2 := goel.NewNamedConcept(1)
	a3 := goel.NewNamedConcept(2)
	r := goel.NewRole(0)
	c := goel.NewELBaseComponents(0, 0, 3, 1)

	id1 := a1.NormalizedID(c)
	id2 := a2.NormalizedID(c)
	id3 := a3.NormalizedID(c)

	gcis := []*goel.GCIConstraint{
		goel.NewGCIConstraint(a1, a2),
		goel.NewGCIConstraint(a2, goel.NewExistentialConcept(r, a3)),
	}

	tbox := goel.NewTBox(c, gcis, nil)
	solver := runConcurrent(tbox)
	assertRContains(t, []goel.BCPair{{id1, id3}, {id2, id3}}, solver.R[uint(r)])
}
