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

package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/FabianWe/goel"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	// builder := goel.RandomELBuilder{NumIndividuals: 10000,
	// 	NumConceptNames:    10000,
	// 	NumRoles:           5000,
	// 	NumConcreteDomains: 0,
	// 	MaxCDSize:          1000,
	// 	MaxNumPredicates:   2000,
	// 	MaxNumFeatures:     1000}
	builder := goel.RandomELBuilder{NumIndividuals: 1000,
		NumConceptNames:    1000,
		NumRoles:           100,
		NumConcreteDomains: 0,
		MaxCDSize:          10,
		MaxNumPredicates:   100,
		MaxNumFeatures:     100}
	fmt.Println("Building random TBox ...")
	_, tbox := builder.GenerateRandomTBox(0, 1000, 100, 5, 100, 100)
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	fmt.Println("Normalizing TBox ...")
	start := time.Now()
	normalized := normalizer.Normalize(tbox)
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	fmt.Println()
	// fmt.Println("==== Naive ===")
	// naive(normalized)
	// fmt.Println()
	fmt.Println("==== Rule Based ===")
	rulebased(normalized)
	fmt.Println()
	fmt.Println("==== Concurrent ====")
	concurrent(normalized)
	fmt.Println()
	fmt.Println("==== Full Concurrent ====")
	fullConcurrent(normalized)
	fmt.Println()
	fmt.Println("==== Transitive Closure ====")
	fullConcurrentTC(normalized)
}

func naive(normalized *goel.NormalizedTBox) {
	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	fmt.Println("Solving ...")
	start := time.Now()
	solver.Solve(normalized)
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
}

func rulebased(normalized *goel.NormalizedTBox) {
	fmt.Println("Building state and rules ...")
	start := time.Now()
	solver := goel.NewAllChangesSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	fmt.Println("Solving ...")
	start = time.Now()
	solver.Solve(normalized)
	execTime = time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
}

func concurrent(normalized *goel.NormalizedTBox) {
	fmt.Println("Building state and rules ...")
	start := time.Now()
	solver := goel.NewConcurrentNotificationSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	fmt.Println("Solving ...")
	start = time.Now()
	solver.Solve(normalized)
	execTime = time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
}

func fullConcurrent(normalized *goel.NormalizedTBox) {
	fmt.Println("Building state and rules ...")
	start := time.Now()
	solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	solver.Workers = 25
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	fmt.Println("Solving ...")
	start = time.Now()
	solver.Solve(normalized)
	execTime = time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
}

func fullConcurrentTC(normalized *goel.NormalizedTBox) {
	fmt.Println("Building state and rules ...")
	start := time.Now()
	solver := goel.NewConcurrentSolver(goel.NewTransitiveClosureGraph(),
		goel.ClosureToSet)
	solver.Init(normalized)
	solver.Workers = 25
	execTime := time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
	fmt.Println("Solving ...")
	start = time.Now()
	solver.Solve(normalized)
	execTime = time.Since(start)
	fmt.Printf("... Done after %v\n", execTime)
}
