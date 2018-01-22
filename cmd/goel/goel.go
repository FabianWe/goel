// Copyright (c) 2016, 2017 Fabian Wenzelmann
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
	"time"

	"gkigit.informatik.uni-freiburg.de/fwenzelmann/goel"
)

func main() {
	// rand.Seed(time.Now().UTC().UnixNano())
	// builder := goel.RandomELBuilder{NumIndividuals: 10000,
	// 	NumConceptNames:    10000,
	// 	NumRoles:           5000,
	// 	NumConcreteDomains: 0,
	// 	MaxCDSize:          1000,
	// 	MaxNumPredicates:   2000,
	// 	MaxNumFeatures:     1000}
	builder := goel.RandomELBuilder{NumIndividuals: 10000,
		NumConceptNames:    100,
		NumRoles:           100,
		NumConcreteDomains: 0,
		MaxCDSize:          10,
		MaxNumPredicates:   100,
		MaxNumFeatures:     100}
	fmt.Println("Building random TBox ...")
	_, tbox := builder.GenerateRandomTBox(0, 1000, 1000, 10, 100, 100)
	fmt.Println("... Done")
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	fmt.Println("Normalizing TBox ...")
	start := time.Now()
	normalized := normalizer.Normalize(tbox)
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	// fmt.Printf("There are %d goroutines running\n", runtime.NumGoroutine())
	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	fmt.Println("Solving ...")
	start = time.Now()
	solver.Solve(normalized)
	execTime = time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)

	baseComp := goel.NewELBaseComponents(0,
		0,
		1,
		3)
	var i uint = 0
	for ; i < baseComp.NumBCD()+1; i++ {
		fmt.Println(baseComp.GetConcept(i))
	}
}
