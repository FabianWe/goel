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

	"github.com/FabianWe/goel/domains"
)

var d = domains.NewRationalDomain()

// TestSat1 tests the constraint f0 = 42.0
func TestSat1(t *testing.T) {
	f0 := domains.NewFeatureID(0)
	r1 := domains.NewEqualsRational(42.0)
	formula1 := domains.NewPredicateFormula(r1, f0)
	res := d.ConjSat(formula1)
	if !res {
		t.Errorf("Formula %v is not satisfiable, expected true.", formula1)
	}
}

// TestSat2 tests if f0 = 5 and f2 > 6 is satisfiable.
func TestSat2(t *testing.T) {
	f0, f1 := domains.NewFeatureID(0), domains.NewFeatureID(1)
	r1 := domains.NewEqualsRational(5)
	r2 := domains.NewGreaterRational(6)
	formula1 := domains.NewPredicateFormula(r1, f0)
	formula2 := domains.NewPredicateFormula(r2, f1)
	res := d.ConjSat(formula1, formula2)
	if !res {
		t.Errorf("Conjunction of %v and %v is not satisfiable, expected true.", formula1, formula2)
	}
}

// TestSat3 tests if f0 = 42, f1 > 21 and f0 = f1 is satisfiable.
func TestSat3(t *testing.T) {
	f0, f1 := domains.NewFeatureID(0), domains.NewFeatureID(1)
	r1 := domains.NewEqualsRational(42.0)
	eq := domains.NewBinaryEqualsRational()
	r2 := domains.NewGreaterRational(21)

	formula1 := domains.NewPredicateFormula(r1, f0)
	formula2 := domains.NewPredicateFormula(eq, f0, f1)
	formula3 := domains.NewPredicateFormula(r2, f1)

	res := d.ConjSat(formula1, formula2, formula3)
	if !res {
		t.Errorf("Conjunction of %v and %v and %v is not satisfiable, expected true",
			formula1, formula2, formula3)
	}
}

// TestSat4 tests if ⊤ is satisfiable.
func TestSat4(t *testing.T) {
	r := domains.NewTrueRational()
	formula := domains.NewPredicateFormula(r)
	res := d.ConjSat(formula)
	if !res {
		t.Error("⊤ is not satisfiable in the rationals, expected true")
	}
}

// TestSat5 tests if f0 = 42 and f0 > 42 is unsatisfiable.
func TestSat5(t *testing.T) {
	f0 := domains.NewFeatureID(0)
	r1 := domains.NewEqualsRational(42)
	r2 := domains.NewGreaterRational(42)

	formula1 := domains.NewPredicateFormula(r1, f0)
	formula2 := domains.NewPredicateFormula(r2, f0)

	res := d.ConjSat(formula1, formula2)
	if res {
		t.Errorf("Conjunction of %v and %v is satisfiable, expected false", formula1, formula2)
	}
}

// TestSat6 tests if f0 = 42, f0 = f1 = f2 = f3 and f3 = 21 is satisfiable.
func TestSat6(t *testing.T) {
	f0, f1, f2, f3 := domains.NewFeatureID(0), domains.NewFeatureID(1), domains.NewFeatureID(2), domains.NewFeatureID(3)

	req1 := domains.NewEqualsRational(42)
	req2 := domains.NewEqualsRational(21)

	r := domains.NewBinaryEqualsRational()

	formula1 := domains.NewPredicateFormula(req1, f0)
	formula2 := domains.NewPredicateFormula(r, f0, f1)
	formula3 := domains.NewPredicateFormula(r, f1, f2)
	formula4 := domains.NewPredicateFormula(r, f2, f3)
	formula5 := domains.NewPredicateFormula(req2, f3)

	res := d.ConjSat(formula1, formula2, formula3, formula4, formula5)
	if res {
		t.Errorf("Conjunction of %v and %v and %v and %v and %v is satisfiable, expected false",
			formula1, formula2, formula3, formula4, formula5)
	}
}
