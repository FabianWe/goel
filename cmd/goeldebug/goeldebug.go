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

	runTest(normalized)
}

func runTest(tbox *goel.NormalizedTBox) {

	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	solver.Solve(tbox)
}
