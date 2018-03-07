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

	"github.com/FabianWe/goel"
	"github.com/FabianWe/goel/domains"
)

func main() {
	// just a single feature and different predicates
	f0 := domains.NewFeatureID(0)
	// predicate1: > 90
	p1 := domains.NewGreaterRational(50.0)
	p2 := domains.NewGreaterRational(40.0)
	p3 := domains.NewGreaterRational(30.0)
	p4 := domains.NewGreaterRational(20.0)

	f1 := domains.NewPredicateFormula(p1, f0)
	f2 := domains.NewPredicateFormula(p2, f0)
	f3 := domains.NewPredicateFormula(p3, f0)
	f4 := domains.NewPredicateFormula(p4, f0)

	ds := domains.NewCDManager()
	ds.Add(domains.NewTypedPredicateFormula(f1, domains.Rationals))
	ds.Add(domains.NewTypedPredicateFormula(f2, domains.Rationals))
	ds.Add(domains.NewTypedPredicateFormula(f3, domains.Rationals))
	ds.Add(domains.NewTypedPredicateFormula(f4, domains.Rationals))

	c := goel.NewELBaseComponents(0, 4, 0, 0)

	tbox := goel.NewTBox(c, nil, nil)
	normalizer := goel.NewDefaultNormalFormBUilder(10)

	normalized := normalizer.Normalize(tbox)

	fmt.Println(ds.Formulae)
	solver := goel.NewNaiveSolver(goel.NewSetGraph(), goel.BFS)
	solver.Solve(normalized, ds)
	fmt.Println(solver.S)
}
