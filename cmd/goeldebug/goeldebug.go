package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/FabianWe/goel"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runTests()
}

var errors uint
var success uint
var unequal uint

func foo() {
	f, openErr := os.Open("debug/compare1.txt")
	if openErr != nil {
		panic(openErr)
	}
	box, readErr := goel.ParseNormalizedTBox(f)
	if readErr != nil {
		panic(readErr)
	}
	for i := 0; i < 1; i++ {
		bar(box)
	}
	fmt.Println(strings.Repeat("=", 20))
}

func runTests() {
	// builder := goel.RandomELBuilder{NumIndividuals: 1000,
	// 	NumConceptNames:    100,
	// 	NumRoles:           100,
	// 	NumConcreteDomains: 0,
	// 	MaxCDSize:          10,
	// 	MaxNumPredicates:   100,
	// 	MaxNumFeatures:     100}
	builder := goel.RandomELBuilder{NumIndividuals: 3,
		NumConceptNames:    3,
		NumRoles:           3,
		NumConcreteDomains: 0,
		MaxCDSize:          10,
		MaxNumPredicates:   100,
		MaxNumFeatures:     100}
	duration := 1 * time.Hour
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

func findD(sets []*goel.BCSet) {
	for i, s := range sets[1:] {
		if s.ContainsID(7) {
			fmt.Println("Found 7 in", i+1)
		}
	}
}

func findR(r1 []*goel.BCPairSet, r2 []*goel.Relation, c, d uint) {
	fmt.Printf("Finding (%d, %d) in r1\n", c, d)
	for i, r := range r1 {
		if r.ContainsID(c, d) {
			fmt.Println("Found in", i)
		}
	}
	fmt.Println(strings.Repeat("=", 20))
	fmt.Printf("Finding (%d, %d) in r2\n", c, d)
	for i, r := range r2 {
		if r.Contains(c, d) {
			fmt.Println("Found in", i)
		}
	}
}

func bar(tbox *goel.NormalizedTBox) {
	s1, r1 := runTest(tbox)
	fmt.Println(strings.Repeat("@", 20))
	s2, r2 := runRuleBased(tbox)
	findR(r1, r2, 10, 13)
	// fmt.Println("7 for s1")
	// findD(s1)
	// fmt.Println("7 for s2")
	// findD(s2)
	res := make(chan bool, 2)
	// compare s and r
	go func() {
		cRes := compareS(s1, s2)
		if !cRes {
			log.Println("Compare of s1 and s2 failed")
		}
		res <- cRes
	}()
	go func() {
		cRes := compareR(r1, r2)
		if !cRes {
			log.Println("Compare of r1 and r2 failed")
		}
		res <- cRes
	}()
	res1 := <-res
	res2 := <-res
	if !(res1 && res2) {
		fmt.Println("FAIL")
	} else {
		fmt.Println("SUCC")
	}
}

func testInstance(builder *goel.RandomELBuilder) {

	// _, tbox := builder.GenerateRandomTBox(0, 1000, 1000, 10, 100, 100)
	_, tbox := builder.GenerateRandomTBox(0, 5, 5, 5, 5, 10)
	normalizer := goel.NewDefaultNormalFormBUilder(100)
	normalized := normalizer.Normalize(tbox)

	defer func() {
		if r := recover(); r != nil {
			errors++
			fmt.Println("====================================")
			fileName := fmt.Sprintf("debug/error%d.txt", errors)
			log.Println("Test failed, writing test to file", fileName)
			file, fErr := os.Create(fileName)
			defer file.Close()
			if fErr != nil {
				log.Println("Writing file failed", fErr)
				return
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
	s2, r2 := runConcurrent(normalized)
	res := make(chan bool, 2)
	// compare s and r
	go func() {
		cRes := compareS(s1, s2)
		if !cRes {
			log.Println("Compare of s1 and s2 failed")
		}
		res <- cRes
	}()
	go func() {
		cRes := compareR(r1, r2)
		if !cRes {
			log.Println("Compare of r1 and r2 failed")
		}
		res <- cRes
	}()
	res1 := <-res
	res2 := <-res
	if !(res1 && res2) {
		unequal++
		fileName := fmt.Sprintf("debug/compare%d.txt", unequal)
		log.Println("Writing failed comp result to file", fileName)
		file, fErr := os.Create(fileName)
		defer file.Close()
		if fErr != nil {
			log.Println("Writing file failed", fErr)
			return
		}
		if writeErr := normalized.Write(file); writeErr != nil {
			log.Println("Writing file failed", writeErr)
		}
		os.Exit(0)
	}
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
		firstRes, secondRes := true, true
		for entry, _ := range next1.M {
			if !next2.ContainsID(entry) {
				fmt.Printf("Missing in s2: %d not in S(%d)\n", entry, i)
				firstRes = false
				break
			}
		}

		for entry, _ := range next2.M {
			if !next1.ContainsID(entry) {
				fmt.Printf("Missing entry in s1: %d not in S(%d)\n", entry, i)
				secondRes = false
				break
			}
		}
		if !(firstRes && secondRes) {
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
		firstRes, secondRes := true, true
		// iterate over each pair in r1 and check if it is contained in r2
		for p, _ := range next1.M {
			if !next2.Contains(p.First, p.Second) {
				fmt.Printf("Missing in r2: r(%d): (%d, %d)\n", i, p.First, p.Second)
				firstRes = false
				break
			}
		}

		// itearte over each entry in r2 and check if it is in r1
		for c, cMap := range next2.Mapping {
			for d, _ := range cMap {
				if !next1.ContainsID(c, d) {
					fmt.Printf("Missing in r1: r(%d): (%d, %d)\n", i, c, d)
					secondRes = false
					break
				}
			}
		}
		if !(firstRes && secondRes) {
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

func runConcurrent(tbox *goel.NormalizedTBox) ([]*goel.BCSet, []*goel.Relation) {
	solver := goel.NewConcurrentNotificationSolver(goel.NewSetGraph(), nil)
	solver.Init(tbox)
	solver.Solve(tbox)
	return solver.S, solver.R
}
