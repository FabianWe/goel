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
	"strings"
	"time"

	"github.com/FabianWe/goel"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var conceptNames, individuals, roles, ci, existentials, ri, numBench int
	flag.IntVar(&conceptNames, "names", 0, "number of concept names")
	flag.IntVar(&individuals, "individuals", 0, "number of individuals")
	flag.IntVar(&roles, "roles", 0, "number of roles")
	flag.IntVar(&ci, "ci", 0, "number of standard ci")
	flag.IntVar(&existentials, "existentials", 0, "number of existentials")
	flag.IntVar(&ri, "ri", 1000, "number of role inclusions")
	flag.IntVar(&numBench, "num", 0, "number of benchmarks to generate")
	var workers int
	flag.IntVar(&workers, "workers", 25, "number of workers for the concurrent solver")
	var cmd, out string
	flag.StringVar(&cmd, "cmd", "", "Command to execute, must be \"run\" or \"build\"")
	flag.StringVar(&out, "out", "", "Directory to print output to / read input from")
	flag.Parse()

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
			log.Printf("Running benchmark \"%s\"\n", fPath)
			// fmt.Println(naive(box))
			fmt.Println(int64(ruleBased(box) / time.Millisecond))
			fmt.Println(int64(notitificationConcurrent(box) / time.Millisecond))
			fmt.Println(int64(concurrent(box, workers) / time.Millisecond))
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

func naive(normalized *goel.NormalizedTBox) time.Duration {
	solver := goel.NewNaiveSolver(
		goel.NewSetGraph(),
		goel.BFS,
	)
	start := time.Now()
	solver.Solve(normalized)
	execTime := time.Since(start)
	return execTime
}

func ruleBased(normalized *goel.NormalizedTBox) time.Duration {
	start := time.Now()
	solver := goel.NewAllChangesSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	solver.Solve(normalized)
	execTime := time.Since(start)
	return execTime
}

func notitificationConcurrent(normalized *goel.NormalizedTBox) time.Duration {
	start := time.Now()
	solver := goel.NewConcurrentNotificationSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	solver.Solve(normalized)
	execTime := time.Since(start)
	return execTime
}

func concurrent(normalized *goel.NormalizedTBox, workers int) time.Duration {
	start := time.Now()
	solver := goel.NewConcurrentSolver(goel.NewSetGraph(), nil)
	solver.Init(normalized)
	solver.Workers = workers
	solver.Solve(normalized)
	execTime := time.Since(start)
	return execTime
}
