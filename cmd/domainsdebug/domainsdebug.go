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
	d := domains.NewRationalDomain()

	f0, f1 := domains.NewFeatureID(0), domains.NewFeatureID(1)
	r1 := domains.NewGreaterRational(42)
	r2 := domains.NewEqualsRational(42)
	r3 := domains.NewEqualsRational(21)

	form1 := domains.NewPredicateFormula(r1, f0)
	form2 := domains.NewPredicateFormula(r2, f0)
	form := domains.NewPredicateFormula(r3, f1)
	res := d.Implies(form, form1, form2)
	fmt.Println(res)
}
