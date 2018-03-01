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

	"github.com/FabianWe/goel/domains"
)

func main() {
	domain := domains.NewRationalDomain()
	f0 := domains.NewFeatureID(0)
	r1 := domains.NewGreaterRational(42.0)
	r2 := domains.NewEqualsRational(42.0)
	formula1 := &domains.PredicateFormula{r1, []domains.FeatureID{f0}}
	formula2 := &domains.PredicateFormula{r2, []domains.FeatureID{f0}}
	lp := domain.FormulateLP(formula1, formula2)
	lp.WriteToStdout()
	fmt.Println(domain.ConjSat(formula1, formula2))
}
