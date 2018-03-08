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

// The problem is as follows: 10 names A1...A10
// 1000 CDExtensions of the form p1 = >(9990), p2 = >(9980), p2 = >(9970)...p1000 = >(0)
// so clearly p1 ⇒ p2 ⇒ ... ⇒ p1000
// also we have Ai ⊑ p1

var benchf0 domains.FeatureID = domains.NewFeatureID(0)
var benchComponents *goel.ELBaseComponents = goel.NewELBaseComponents(0, 100, 10, 0)
var benchNormalized *goel.NormalizedTBox
var benchds *domains.CDManager = domains.NewCDManager()

// dummy variables to prevent compiler optimisation
var benchDummy int

func init() {
	for i := 0; i < 100; i++ {
		p := domains.NewGreaterRational(float64(9990 - (i * 10)))
		f := domains.NewPredicateFormula(p, benchf0)
		benchds.Add(domains.NewTypedPredicateFormula(f, domains.Rationals))
	}
	gcis := make([]*goel.GCIConstraint, 0)
	firstFormula := goel.NewConcreteDomainExtension(0)
	for i := 0; i < 2; i++ {
		gcis = append(gcis, goel.NewGCIConstraint(goel.NewNamedConcept(uint(i)), firstFormula))
	}
	tbox := goel.NewTBox(benchComponents, gcis, nil)
	normalizer := goel.NewDefaultNormalFormBUilder(10)
	benchNormalized = normalizer.Normalize(tbox)
}

// func BenchmarkNaiveCD(b *testing.B) {
// 	for n := 0; n < b.N; n++ {
// 		solver := goel.NewNaiveSolver(goel.NewSetGraph(), goel.BFS)
// 		solver.Solve(benchNormalized, benchds)
// 	}
// }

func BenchmarkRuleBaseCD(b *testing.B) {
	for n := 0; n < b.N; n++ {
		solver := goel.NewAllChangesSolver(goel.NewSetGraph(), nil)
		solver.Init(benchNormalized, benchds)
		solver.Solve(benchNormalized)
	}
}

func BenchmarkNotificationConcurrentCD(b *testing.B) {
	for n := 0; n < b.N; n++ {
		solver := goel.NewConcurrentNotificationSolver(goel.NewSetGraph(), nil)
		solver.Init(benchNormalized, benchds)
		solver.Solve(benchNormalized)
	}
}

func BenchmarkFullConcurrentCD(b *testing.B) {
	for n := 0; n < b.N; n++ {
		solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
		solver.Init(benchNormalized, benchds)
		solver.Solve(benchNormalized)
	}
}

// func TestNaiveCD(t *testing.T) {
// 	solver := goel.NewNaiveSolver(goel.NewSetGraph(), goel.BFS)
// 	solver.Solve(benchNormalized, benchds)
// }
