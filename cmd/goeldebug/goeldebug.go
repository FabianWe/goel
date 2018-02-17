package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/FabianWe/goel"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runTests()
}

var errors uint
var success uint

func foo() {
	f, openErr := os.Open("debug/error1.txt")
	if openErr != nil {
		panic(openErr)
	}
	box, readErr := goel.ParseNormalizedTBox(f)
	if readErr != nil {
		panic(readErr)
	}
	runTest(box)
}

func runTests() {
	builder := goel.RandomELBuilder{NumIndividuals: 10000,
		NumConceptNames:    100,
		NumRoles:           100,
		NumConcreteDomains: 0,
		MaxCDSize:          10,
		MaxNumPredicates:   100,
		MaxNumFeatures:     100}
	duration := 5 * time.Hour
	start := time.Now()
	for {
		expired := time.Since(start)
		if expired >= duration {
			fmt.Println("Done with all tests.")
			fmt.Printf("Success: %d, Errors: %d", success, errors)
			return
		} else {
			testInstance(&builder)
		}
	}
}

func testInstance(builder *goel.RandomELBuilder) {

	_, tbox := builder.GenerateRandomTBox(0, 1000, 1000, 10, 100, 100)
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	normalized := normalizer.Normalize(tbox)

	defer func() {
		if r := recover(); r != nil {
			errors++
			fmt.Println("====================================")
			fileName := fmt.Sprintf("debug/error%d.txt", errors)
			log.Println("Test failed, writing test to file", fileName)
			file, fErr := os.Create(fileName)
			if fErr != nil {
				log.Println("Writing file failed", fErr)
			}
			if writeErr := normalized.Write(file); writeErr != nil {
				log.Println("Writing file failed", writeErr)
			}
			log.Println("TBox BCC-Components:", tbox.Components.NumBCD())
			log.Println("Normalized TBox BCC-Components:", normalized.Components.NumBCD())
		} else {
			success++
		}
	}()
	s1, r1 := runTest(normalized)
	s2, r2 := runRuleBased(normalized)
	done := make(chan bool, 2)
	// compare s and r
	go func() {
		if !compareS(s1, s2) {
			log.Println("Compare of s1 and s2 failed")
		}
		done <- true
	}()
	go func() {
		if !compareR(r1, r2) {
			log.Println("Compare of r1 and r2 failed")
		}
		done <- true
	}()
	<-done
	<-done
}

// TODO not nice, just there until it is defined in a more common way
func compareS(s1, s2 []*goel.BCSet) bool {
	if len(s1) != len(s2) {
		log.Println("S mapping for the tbox are not of the same length!")
		return false
	}
	n := len(s1)
	for i := 1; i < n; i++ {
		next1 := s1[i]
		next2 := s2[i]
		res := make(chan bool, 2)
		go func() {
			res <- next1.IsSubset(next2)
		}()
		go func() {
			res <- next2.IsSubset(next1)
		}()
		res1 := <-res
		res2 := <-res
		if !(res1 && res2) {
			return false
		}
	}
	return true
}

func compareR(r1 []*goel.BCPairSet, r2 []*goel.Relation) bool {
	if len(r1) != len(r2) {
		log.Println("R mapping for the tbox are not of the same length")
		return false
	}
	n := len(r1)
	for i := 0; i < n; i++ {
		next1 := r1[i]
		next2 := r2[i]
		res := make(chan bool, 2)
		// iterate over each pair in r1 and check if it is contained in r2
		go func() {
			for p, _ := range next1.M {
				if !next2.Contains(p.First, p.Second) {
					res <- false
					return
				}
			}
			res <- true
		}()
		// itearte over each entry in r2 and check if it is in r1
		go func() {
			for c, cMap := range next2.Mapping {
				for d, _ := range cMap {
					if !next1.ContainsID(c, d) {
						res <- false
						return
					}
				}
			}
			res <- true
		}()
		res1 := <-res
		res2 := <-res
		if !res1 && res2 {
			return false
		}
	}
	return true
}

func runTest(tbox *goel.NormalizedTBox) ([]*goel.BCSet, []*goel.BCPairSet) {

	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	solver.Solve(tbox)
	return solver.S, solver.R
}

func runRuleBased(tbox *goel.NormalizedTBox) ([]*goel.BCSet, []*goel.Relation) {
	solver := goel.NewAllChangesSolver(goel.NewSetGraph(), nil)
	solver.Init(tbox)
	solver.Solve(tbox)
	return solver.S, solver.R
}
