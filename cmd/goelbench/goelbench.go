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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/FabianWe/goel"
	"github.com/FabianWe/goel/domains"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var conceptNames, individuals, roles, ci, existentials, ri, numBench int
	var compare bool

	flag.IntVar(&conceptNames, "names", 0, "number of concept names")
	flag.IntVar(&individuals, "individuals", 0, "number of individuals")
	flag.IntVar(&roles, "roles", 0, "number of roles")
	flag.IntVar(&ci, "ci", 0, "number of standard ci")
	flag.IntVar(&existentials, "existentials", 0, "number of existentials")
	flag.IntVar(&ri, "ri", 1000, "number of role inclusions")
	flag.IntVar(&numBench, "num", 0, "number of benchmarks to generate")
	flag.BoolVar(&compare, "compare", false, "if true the results of solver will be compared")
	var workers int
	flag.IntVar(&workers, "workers", 25, "number of workers for the concurrent solver")
	var cmd, out string
	var cpuProfile string
	flag.StringVar(&cmd, "cmd", "", "Command to execute, must be \"run\" or \"build\"")
	flag.StringVar(&out, "out", "", "Directory to print output to / read input from")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if cpuProfile != "" {
		profileFile, profileErr := os.Create(cpuProfile)
		if profileErr != nil {
			log.Fatal(profileErr)
		}
		pprof.StartCPUProfile(profileFile)
		defer pprof.StopCPUProfile()

	}

	switch cmd {
	case "run":
		if out == "" {
			fmt.Println("benchmark directory not specified via -out PATH")
			os.Exit(1)
		}
		log.Printf("Reading benchmarks from \"%s\"\n", out)
		files, dirErr := ioutil.ReadDir(out)
		if dirErr != nil {
			log.Println("Unable to read directory:", dirErr)
			os.Exit(1)
		}
		benchFiles := make([]string, 0, len(files))
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if strings.HasSuffix(f.Name(), ".txt") {
				benchFiles = append(benchFiles, f.Name())
			}
		}
		for _, f := range benchFiles {
			lastRMapping = nil
			currentRMapping = nil
			lastSMapping = nil
			currentSMapping = nil
			fPath := path.Join(out, f)
			file, openErr := os.Open(fPath)
			if openErr != nil {
				log.Printf("Unable to open file \"%s\":\n", fPath)
				log.Println(openErr)
			}
			box, readErr := goel.ParseNormalizedTBox(file)
			file.Close()
			if readErr != nil {
				log.Printf("Can't parse TBox from \"%s\":\n", fPath)
				log.Println(readErr)
			}
			// TODO domains created here, not so nice
			domains := domains.NewCDManager()
			log.Printf("Running benchmark \"%s\"\n", fPath)

			runtime.GC()
			fmt.Println("Rule Based NC          ", int64(rbnc(box, domains)/time.Millisecond))
			if compare {
				compareMappings()
			}

			// runtime.GC()
			// fmt.Println("Rule Based             ", int64(ruleBased(box, domains)/time.Millisecond))
			// if compare {
			// 	compareMappings()
			// }
			runtime.GC()
			fmt.Println("Full Concurrent        ", int64(concurrent(box, domains, workers)/time.Millisecond))
			if compare {
				compareMappings()
			}
			runtime.GC()
			fmt.Println("Bulk concurrent        ", int64(bulk(box, domains, workers)/time.Millisecond))
			if compare {
				compareMappings()
			}
		}
	case "build":
		if out == "" {
			fmt.Println("output directory not specified via -out PATH")
			os.Exit(1)
		}
		log.Printf("Building %d benchmarks, saving to \"%s\"\n", numBench, out)
		// builder := goel.RandomELBuilder{
		// 	NumIndividuals:     0,
		// 	NumConceptNames:    conceptNames,
		// 	NumRoles:           roles,
		// 	NumConcreteDomains: 0,
		// 	MaxCDSize:          0,
		// 	MaxNumPredicates:   0,
		// 	MaxNumFeatures:     0,
		// }
		builder := goel.NormalizedRandomELBuilder{
			NumIndividuals:  uint(individuals),
			NumConceptNames: uint(conceptNames),
			NumRoles:        uint(roles),
		}
		var i int
		for ; i < numBench; i++ {
			// _, tbox := builder.GenerateRandomTBox(0, conjunctions, existentials, riLHS, gci, ri)
			// normalizer := goel.NewDefaultNormalFormBUilder(100)
			// normalized := normalizer.Normalize(tbox)
			normalized := builder.GenerateRandomTBox(ci, existentials, ri)
			// create file
			fileName := fmt.Sprintf("%s/bench%d.txt", out, i)
			file, fErr := os.Create(fileName)
			if fErr != nil {
				log.Println("Writing file failed", fErr)
				continue
			}
			if writeErr := normalized.Write(file); writeErr != nil {
				log.Println("Writing file failed", writeErr)
				file.Close()
				continue
			} else {
				file.Close()
			}
		}
		log.Println("Done creating benchmarks")
	default:
		fmt.Println("cmd invalid / not specified, must be \"run\" or \"build\"")
		os.Exit(1)
	}
}

func naive(normalized *goel.NormalizedTBox, domains *domains.CDManager) time.Duration {
	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	start := time.Now()
	solver.Solve(normalized, domains)
	execTime := time.Since(start)
	return execTime
}

var lastSMapping, currentSMapping []*goel.BCSet
var lastRMapping, currentRMapping []*goel.Relation

func shiftMapping(newS []*goel.BCSet, newR []*goel.Relation) {
	lastSMapping = currentSMapping
	lastRMapping = currentRMapping

	currentSMapping = newS
	currentRMapping = newR
}

func compareMappings() {
	if lastSMapping == nil || lastRMapping == nil || currentSMapping == nil || currentRMapping == nil {
		return
	}
	cmp1 := goel.CompareSMapping(lastSMapping, currentSMapping)
	if !cmp1 {
		log.Println("Compare of S mappings failed")
	}
	cmp2 := goel.CompareRMapping(lastRMapping, currentRMapping)
	if !cmp2 {
		log.Println("Compare of R mappings failed")
	}
}

func ruleBased(normalized *goel.NormalizedTBox, domains *domains.CDManager) time.Duration {
	start := time.Now()
	solver := goel.NewAllChangesSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized, domains)
	solver.Solve(normalized)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}

func notitificationConcurrent(normalized *goel.NormalizedTBox, domains *domains.CDManager) time.Duration {
	start := time.Now()
	solver := goel.NewConcurrentNotificationSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized, domains)
	solver.Solve(normalized)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}

func concurrent(normalized *goel.NormalizedTBox, domains *domains.CDManager, workers int) time.Duration {
	start := time.Now()
	solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
	solver.Workers = workers
	solver.Init(normalized, domains)
	solver.Solve(normalized)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}

func fullConcurrentTC(tbox *goel.NormalizedTBox, domains *domains.CDManager, workers int) time.Duration {
	start := time.Now()
	solver := goel.NewConcurrentSolver(goel.NewTransitiveClosureGraph(),
		goel.ClosureToSet)
	solver.Workers = workers
	solver.Init(tbox, domains)
	solver.Solve(tbox)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}

func bulk(tbox *goel.NormalizedTBox, domains *domains.CDManager, workers int) time.Duration {
	start := time.Now()
	solver := goel.NewBulkSolver(goel.NewSetGraph(), nil)
	solver.Workers = workers
	solver.K = 200
	solver.Init(tbox, domains)
	solver.Solve(tbox)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}

func rbnc(tbox *goel.NormalizedTBox, domains *domains.CDManager) time.Duration {
	start := time.Now()
	solver := goel.NewNCRBSolver(goel.NewSetGraph(), nil)
	solver.Init(tbox, domains)
	solver.Solve(tbox)
	execTime := time.Since(start)
	shiftMapping(solver.S, solver.R)
	return execTime
}
