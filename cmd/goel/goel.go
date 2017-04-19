// The MIT License (MIT)

// Copyright (c) 2016, 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

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
	"math/rand"
	"time"

	"gkigit.informatik.uni-freiburg.de/fwenzelmann/goel"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	builder := goel.RandomELBuilder{NumIndividuals: 10000,
		NumConceptNames:    10000,
		NumRoles:           5000,
		NumConcreteDomains: 100,
		MaxCDSize:          1000,
		MaxNumPredicates:   2000,
		MaxNumFeatures:     1000}
	fmt.Println("Building random TBox ...")
	start := time.Now()
	_, tbox := builder.GenerateRandomTBox(1000, 10000, 10000, 100, 10000, 10000)
	end := time.Since(start)
	fmt.Printf("... Done after %v\n\n", end)
	fmt.Println(tbox.ApplyPhaseOne(1))
}
